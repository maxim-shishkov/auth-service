package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maxim-shishkov/auth-service/internal/domain"
	transporthttp "github.com/maxim-shishkov/auth-service/internal/transport/http"
)

func newTestRouter(svc *fakeService) http.Handler {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := transporthttp.NewHandler(svc, log)
	return transporthttp.NewRouter(h, fakeTokenParser{}, log)
}

func doRequest(t *testing.T, router http.Handler, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(b)
	}

	req := httptest.NewRequest(method, path, reader)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}

func TestRegister_Success(t *testing.T) {
	svc := &fakeService{
		registerFn: func(_ context.Context, email, password string) (*domain.User, error) {
			assert.Equal(t, "user@example.com", email)
			assert.Equal(t, "strong-password", password)
			return fixedUser, nil
		},
	}
	router := newTestRouter(svc)

	rec := doRequest(t, router, http.MethodPost, "/register",
		map[string]string{"email": "user@example.com", "password": "strong-password"}, nil)

	require.Equal(t, http.StatusCreated, rec.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, fixedUser.Email, resp["email"])
	assert.NotContains(t, rec.Body.String(), "password", "ответ не должен содержать пароль/хеш")
}

func TestRegister_InvalidData(t *testing.T) {
	svc := &fakeService{
		registerFn: func(_ context.Context, _, _ string) (*domain.User, error) {
			return nil, domain.ErrInvalidEmail
		},
	}
	router := newTestRouter(svc)

	rec := doRequest(t, router, http.MethodPost, "/register",
		map[string]string{"email": "not-an-email", "password": "strong-password"}, nil)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegister_MalformedJSON(t *testing.T) {
	svc := &fakeService{}
	router := newTestRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(`{not-json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc := &fakeService{
		registerFn: func(_ context.Context, _, _ string) (*domain.User, error) {
			return nil, domain.ErrUserAlreadyExists
		},
	}
	router := newTestRouter(svc)

	rec := doRequest(t, router, http.MethodPost, "/register",
		map[string]string{"email": "dup@example.com", "password": "strong-password"}, nil)

	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestLogin_Success(t *testing.T) {
	svc := &fakeService{
		loginFn: func(_ context.Context, email, password string) (string, error) {
			return "signed-token", nil
		},
	}
	router := newTestRouter(svc)

	rec := doRequest(t, router, http.MethodPost, "/login",
		map[string]string{"email": "user@example.com", "password": "strong-password"}, nil)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "signed-token", resp["token"])
}

func TestLogin_WrongCredentials(t *testing.T) {
	svc := &fakeService{
		loginFn: func(_ context.Context, _, _ string) (string, error) {
			return "", domain.ErrInvalidCredentials
		},
	}
	router := newTestRouter(svc)

	rec := doRequest(t, router, http.MethodPost, "/login",
		map[string]string{"email": "user@example.com", "password": "wrong"}, nil)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMe_ValidAuth(t *testing.T) {
	svc := &fakeService{
		meFn: func(_ context.Context, userID uuid.UUID) (*domain.User, error) {
			assert.Equal(t, fixedUser.ID, userID)
			return fixedUser, nil
		},
	}
	router := newTestRouter(svc)

	rec := doRequest(t, router, http.MethodGet, "/me", nil, map[string]string{
		"Authorization": "Bearer valid-token-" + fixedUser.ID.String(),
	})

	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, fixedUser.Email, resp["email"])
	assert.NotContains(t, rec.Body.String(), "password")
}

func TestMe_NoAuthHeader(t *testing.T) {
	router := newTestRouter(&fakeService{})

	rec := doRequest(t, router, http.MethodGet, "/me", nil, nil)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMe_MalformedAuthHeader(t *testing.T) {
	router := newTestRouter(&fakeService{})

	rec := doRequest(t, router, http.MethodGet, "/me", nil, map[string]string{
		"Authorization": "not-a-bearer-token",
	})

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMe_ExpiredToken(t *testing.T) {
	router := newTestRouter(&fakeService{})

	rec := doRequest(t, router, http.MethodGet, "/me", nil, map[string]string{
		"Authorization": "Bearer expired-token",
	})

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMe_InvalidToken(t *testing.T) {
	router := newTestRouter(&fakeService{})

	rec := doRequest(t, router, http.MethodGet, "/me", nil, map[string]string{
		"Authorization": "Bearer garbage",
	})

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_PanicIsRecovered(t *testing.T) {
	svc := &fakeService{
		registerFn: func(_ context.Context, _, _ string) (*domain.User, error) {
			panic("boom")
		},
	}
	router := newTestRouter(svc)

	rec := doRequest(t, router, http.MethodPost, "/register",
		map[string]string{"email": "user@example.com", "password": "strong-password"}, nil)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

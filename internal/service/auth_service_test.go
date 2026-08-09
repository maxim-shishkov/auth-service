package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maxim-shishkov/auth-service/internal/domain"
	"github.com/maxim-shishkov/auth-service/internal/service"
)

func newService() (*service.AuthService, *fakeRepo, *fakeTokenManager) {
	repo := newFakeRepo()
	tok := &fakeTokenManager{}
	return service.NewAuthService(repo, tok), repo, tok
}

func TestAuthService_Register_Success(t *testing.T) {
	s, repo, _ := newService()

	u, err := s.Register(context.Background(), "user@example.com", "strong-password")
	require.NoError(t, err)
	assert.Equal(t, "user@example.com", u.Email)
	assert.NotEmpty(t, u.PasswordHash)
	assert.NotEqual(t, "strong-password", u.PasswordHash, "пароль не должен храниться в открытом виде")
	assert.NotEqual(t, uuid.Nil, u.ID)

	stored, err := repo.GetByEmail(context.Background(), "user@example.com")
	require.NoError(t, err)
	assert.Equal(t, u.ID, stored.ID)
}

func TestAuthService_Register_InvalidEmail(t *testing.T) {
	s, _, _ := newService()

	cases := []string{"", "not-an-email", "foo@", "@bar.com", "foo bar@baz.com"}
	for _, email := range cases {
		email := email
		t.Run(email, func(t *testing.T) {
			_, err := s.Register(context.Background(), email, "strong-password")
			assert.ErrorIs(t, err, domain.ErrInvalidEmail)
		})
	}
}

func TestAuthService_Register_InvalidPassword(t *testing.T) {
	s, _, _ := newService()

	_, err := s.Register(context.Background(), "user@example.com", "short")
	assert.ErrorIs(t, err, domain.ErrInvalidPassword)
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	s, _, _ := newService()
	ctx := context.Background()

	_, err := s.Register(ctx, "dup@example.com", "strong-password")
	require.NoError(t, err)

	_, err = s.Register(ctx, "dup@example.com", "another-password")
	assert.ErrorIs(t, err, domain.ErrUserAlreadyExists)
}

func TestAuthService_Register_RepositoryFailure(t *testing.T) {
	s, repo, _ := newService()
	repo.failNext = errors.New("boom")

	_, err := s.Register(context.Background(), "user@example.com", "strong-password")
	require.Error(t, err)
	assert.False(t, errors.Is(err, domain.ErrUserAlreadyExists))
}

func TestAuthService_Login_Success(t *testing.T) {
	s, _, _ := newService()
	ctx := context.Background()

	_, err := s.Register(ctx, "user@example.com", "strong-password")
	require.NoError(t, err)

	tok, err := s.Login(ctx, "user@example.com", "strong-password")
	require.NoError(t, err)
	assert.NotEmpty(t, tok)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	s, _, _ := newService()
	ctx := context.Background()

	_, err := s.Register(ctx, "user@example.com", "strong-password")
	require.NoError(t, err)

	_, err = s.Login(ctx, "user@example.com", "wrong-password")
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestAuthService_Login_UnknownEmail(t *testing.T) {
	s, _, _ := newService()

	_, err := s.Login(context.Background(), "ghost@example.com", "whatever-password")
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials,
		"ошибка должна быть той же, что и для неверного пароля — нельзя раскрывать существование email")
}

func TestAuthService_Login_TokenGenerationFailure(t *testing.T) {
	s, _, tok := newService()
	ctx := context.Background()

	_, err := s.Register(ctx, "user@example.com", "strong-password")
	require.NoError(t, err)

	tok.failNext = errors.New("signing key unavailable")

	_, err = s.Login(ctx, "user@example.com", "strong-password")
	require.Error(t, err)
}

func TestAuthService_Me_Success(t *testing.T) {
	s, _, _ := newService()
	ctx := context.Background()

	created, err := s.Register(ctx, "user@example.com", "strong-password")
	require.NoError(t, err)

	got, err := s.Me(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.Email, got.Email)
}

func TestAuthService_Me_NotFound(t *testing.T) {
	s, _, _ := newService()

	_, err := s.Me(context.Background(), uuid.New())
	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}

func BenchmarkLogin_Timing(b *testing.B) {
	s, _, _ := newService()
	ctx := context.Background()

	_, err := s.Register(ctx, "user@example.com", "strong-password")
	if err != nil {
		b.Fatal(err)
	}

	b.Run("existing-user-wrong-password", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = s.Login(
				ctx,
				"user@example.com",
				"wrong-password",
			)
		}
	})

	b.Run("unknown-user", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = s.Login(
				ctx,
				"unknown@example.com",
				"wrong-password",
			)
		}
	})
}

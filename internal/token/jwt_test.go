package token_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/maxim-shishkov/auth-service/internal/domain"
	"github.com/maxim-shishkov/auth-service/internal/token"
)

func TestManager_GenerateAndParse_Success(t *testing.T) {
	m := token.NewManager("test-secret", time.Minute)
	userID := uuid.New()

	tok, err := m.Generate(userID)
	require.NoError(t, err)
	require.NotEmpty(t, tok)

	gotID, err := m.Parse(tok)
	require.NoError(t, err)
	require.Equal(t, userID, gotID)
}

func TestManager_Parse_Expired(t *testing.T) {
	m := token.NewManager("test-secret", -time.Minute) // уже истёкший TTL
	userID := uuid.New()

	tok, err := m.Generate(userID)
	require.NoError(t, err)

	_, err = m.Parse(tok)
	require.ErrorIs(t, err, domain.ErrTokenExpired)
}

func TestManager_Parse_InvalidSignature(t *testing.T) {
	m1 := token.NewManager("secret-one", time.Minute)
	m2 := token.NewManager("secret-two", time.Minute)

	tok, err := m1.Generate(uuid.New())
	require.NoError(t, err)

	_, err = m2.Parse(tok)
	require.ErrorIs(t, err, domain.ErrTokenInvalid)
}

func TestManager_Parse_Garbage(t *testing.T) {
	m := token.NewManager("secret", time.Minute)

	_, err := m.Parse("not-a-jwt-token")
	require.ErrorIs(t, err, domain.ErrTokenInvalid)
}

func TestManager_Parse_Empty(t *testing.T) {
	m := token.NewManager("secret", time.Minute)

	_, err := m.Parse("")
	require.Error(t, err)
}

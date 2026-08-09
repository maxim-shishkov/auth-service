package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/maxim-shishkov/auth-service/internal/domain"
)

type Manager struct {
	secret []byte
	ttl    time.Duration
}

func NewManager(secret string, ttl time.Duration) *Manager {
	return &Manager{secret: []byte(secret), ttl: ttl}
}

type claims struct {
	UserID uuid.UUID `json:"uid"`
	jwt.RegisteredClaims
}

func (m *Manager) Generate(userID uuid.UUID) (string, error) {
	now := time.Now()

	c := claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "auth-service",
			Audience:  jwt.ClaimStrings{"api"},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
			Subject:   userID.String(),
		},
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, c)

	signed, err := tok.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("token: sign: %w", err)
	}

	return signed, nil
}

func (m *Manager) Parse(tokenString string) (uuid.UUID, error) {
	c := &claims{}

	tok, err := jwt.ParseWithClaims(tokenString, c, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])

		}

		return m.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return uuid.Nil, domain.ErrTokenExpired
		}

		return uuid.Nil, domain.ErrTokenInvalid
	}

	if !tok.Valid {
		return uuid.Nil, domain.ErrTokenInvalid
	}

	if c.UserID == uuid.Nil {
		return uuid.Nil, domain.ErrTokenInvalid

	}

	return c.UserID, nil
}

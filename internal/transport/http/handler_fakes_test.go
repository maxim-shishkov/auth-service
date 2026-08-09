package http_test

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/maxim-shishkov/auth-service/internal/domain"
)

type fakeService struct {
	registerFn func(ctx context.Context, email, password string) (*domain.User, error)
	loginFn    func(ctx context.Context, email, password string) (string, error)
	meFn       func(ctx context.Context, userID uuid.UUID) (*domain.User, error)
}

func (f *fakeService) Register(ctx context.Context, email, password string) (*domain.User, error) {
	return f.registerFn(ctx, email, password)
}

func (f *fakeService) Login(ctx context.Context, email, password string) (string, error) {
	return f.loginFn(ctx, email, password)
}

func (f *fakeService) Me(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	return f.meFn(ctx, userID)
}

type fakeTokenParser struct{}

func (fakeTokenParser) Parse(tokenString string) (uuid.UUID, error) {
	const validPrefix = "valid-token-"

	switch {
	case tokenString == "expired-token":
		return uuid.Nil, domain.ErrTokenExpired
	case len(tokenString) > len(validPrefix) && tokenString[:len(validPrefix)] == validPrefix:
		id, err := uuid.Parse(tokenString[len(validPrefix):])
		if err != nil {
			return uuid.Nil, domain.ErrTokenInvalid
		}
		return id, nil
	default:
		return uuid.Nil, domain.ErrTokenInvalid
	}
}

var fixedUser = &domain.User{
	ID:        uuid.MustParse("11111111-1111-1111-1111-111111111111"),
	Email:     "user@example.com",
	CreatedAt: time.Now(),
}

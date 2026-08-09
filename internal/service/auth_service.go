package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/maxim-shishkov/auth-service/internal/domain"
)

const minPasswordLength = 8

type TokenManager interface {
	Generate(userID uuid.UUID) (string, error)
}

type UserRepository interface {
	Create(ctx context.Context, u *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

type AuthService struct {
	repo  UserRepository
	token TokenManager
	now   func() time.Time
}

func NewAuthService(repo UserRepository, token TokenManager) *AuthService {
	return &AuthService{repo: repo, token: token, now: time.Now}
}

func (s *AuthService) Register(ctx context.Context, email, password string) (*domain.User, error) {
	email, err := normalizeAndValidateEmail(email)
	if err != nil {
		return nil, fmt.Errorf("normalize and validateEmail: %w", err)
	}

	if err := validatePassword(password); err != nil {
		return nil, fmt.Errorf("validate password: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("service: hash password: %w", err)
	}

	u := &domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hash),
		CreatedAt:    s.now().UTC(),
	}

	if err := s.repo.Create(ctx, u); err != nil {
		if errors.Is(err, domain.ErrUserAlreadyExists) {
			return nil, domain.ErrUserAlreadyExists
		}

		return nil, fmt.Errorf("service: create user: %w", err)
	}

	return u, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	email, err := normalizeAndValidateEmail(email)
	if err != nil {
		return "", domain.ErrInvalidCredentials
	}

	u, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			_ = bcrypt.CompareHashAndPassword([]byte(dummyBcryptHash), []byte(password))

			return "", domain.ErrInvalidCredentials
		}

		return "", fmt.Errorf("service: get user by email: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", domain.ErrInvalidCredentials
	}

	tok, err := s.token.Generate(u.ID)
	if err != nil {
		return "", fmt.Errorf("service: generate token: %w", err)
	}

	return tok, nil
}

func (s *AuthService) Me(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrUserNotFound
		}

		return nil, fmt.Errorf("service: get user by id: %w", err)
	}

	return u, nil
}

func normalizeAndValidateEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return "", domain.ErrInvalidEmail
	}

	return addr.Address, nil
}

func validatePassword(password string) error {
	if len(password) < minPasswordLength {
		return domain.ErrInvalidPassword
	}

	return nil
}

const dummyBcryptHash = "$2a$10$CwTycUXWue0Thq9StjUM0uJ8G8b8V.4qDL7mZ8mFqU4mCE2WkYy0G"

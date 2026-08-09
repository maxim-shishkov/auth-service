package service_test

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/maxim-shishkov/auth-service/internal/domain"
)

type fakeRepo struct {
	mu       sync.Mutex
	byID     map[uuid.UUID]*domain.User
	byEmail  map[string]*domain.User
	failNext error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		byID:    make(map[uuid.UUID]*domain.User),
		byEmail: make(map[string]*domain.User),
	}
}

func (f *fakeRepo) Create(_ context.Context, u *domain.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.failNext != nil {
		err := f.failNext
		f.failNext = nil
		return err
	}

	if _, exists := f.byEmail[u.Email]; exists {
		return domain.ErrUserAlreadyExists
	}

	cp := *u
	f.byID[u.ID] = &cp
	f.byEmail[u.Email] = &cp

	return nil
}

func (f *fakeRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	u, ok := f.byEmail[email]
	if !ok {
		return nil, domain.ErrUserNotFound
	}

	cp := *u
	return &cp, nil
}

func (f *fakeRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	u, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}

	cp := *u
	return &cp, nil
}

type fakeTokenManager struct {
	failNext error
}

func (f *fakeTokenManager) Generate(userID uuid.UUID) (string, error) {
	if f.failNext != nil {
		err := f.failNext
		f.failNext = nil
		return "", err
	}

	return "token-" + userID.String(), nil
}

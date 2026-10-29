package auth

import (
	"context"
	"sync"
)

type memStore struct {
	mu      sync.Mutex
	byID    map[int64]User
	byEmail map[string]User
}

func newMemStore() *memStore {
	return &memStore{
		byID:    map[int64]User{},
		byEmail: map[string]User{},
	}
}

func (m *memStore) Insert(_ context.Context, u User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.byEmail[u.Email]; ok {
		return ErrEmailTaken
	}
	cp := u
	m.byID[u.ID] = cp
	m.byEmail[u.Email] = cp
	return nil
}

func (m *memStore) ByEmail(_ context.Context, email string) (User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.byEmail[email]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

func (m *memStore) ByID(_ context.Context, id int64) (User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.byID[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

package auth

import (
	"bytes"
	"context"
	"sync"
	"time"
)

type memToken struct {
	ID        int64
	UserID    int64
	Purpose   string
	Hash      []byte
	ExpiresAt time.Time
	Consumed  *time.Time
}

type memStore struct {
	mu      sync.Mutex
	byID    map[int64]User
	byEmail map[string]User
	tokens  []memToken
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

func (m *memStore) IssueToken(_ context.Context, tok AuthToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := tok.CreatedAt
	for i := range m.tokens {
		t := &m.tokens[i]
		if t.UserID == tok.UserID && t.Purpose == tok.Purpose && t.Consumed == nil {
			c := now
			t.Consumed = &c
		}
	}
	m.tokens = append(m.tokens, memToken{
		ID:        tok.ID,
		UserID:    tok.UserID,
		Purpose:   tok.Purpose,
		Hash:      append([]byte(nil), tok.Hash...),
		ExpiresAt: tok.ExpiresAt,
	})
	return nil
}

func (m *memStore) ConsumeVerifyToken(_ context.Context, hash []byte, now time.Time) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.tokens {
		t := &m.tokens[i]
		if !bytes.Equal(t.Hash, hash) || t.Purpose != PurposeVerifyEmail || t.Consumed != nil {
			continue
		}
		if !t.ExpiresAt.After(now) {
			continue
		}
		c := now
		t.Consumed = &c
		u, ok := m.byID[t.UserID]
		if ok {
			u.EmailVerified = true
			m.byID[u.ID] = u
			m.byEmail[u.Email] = u
		}
		return t.UserID, nil
	}
	return 0, ErrInvalidToken
}

func (m *memStore) LiveResetUser(_ context.Context, hash []byte, now time.Time) (User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.tokens {
		t := &m.tokens[i]
		if !bytes.Equal(t.Hash, hash) || t.Purpose != PurposeResetPassword || t.Consumed != nil {
			continue
		}
		if !t.ExpiresAt.After(now) {
			continue
		}
		u, ok := m.byID[t.UserID]
		if !ok {
			return User{}, ErrInvalidResetToken
		}
		return u, nil
	}
	return User{}, ErrInvalidResetToken
}

func (m *memStore) ConsumeResetAndSetPassword(_ context.Context, hash []byte, passwordHash string, now time.Time) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.tokens {
		t := &m.tokens[i]
		if !bytes.Equal(t.Hash, hash) || t.Purpose != PurposeResetPassword || t.Consumed != nil {
			continue
		}
		if !t.ExpiresAt.After(now) {
			continue
		}
		c := now
		t.Consumed = &c
		u, ok := m.byID[t.UserID]
		if !ok {
			return 0, ErrInvalidResetToken
		}
		u.PasswordHash = passwordHash
		m.byID[u.ID] = u
		m.byEmail[u.Email] = u
		return t.UserID, nil
	}
	return 0, ErrInvalidResetToken
}

func (m *memStore) UpdatePasswordHash(_ context.Context, userID int64, passwordHash string, now time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.byID[userID]
	if !ok {
		return ErrNotFound
	}
	u.PasswordHash = passwordHash
	_ = now
	m.byID[u.ID] = u
	m.byEmail[u.Email] = u
	return nil
}

func (m *memStore) unusedCount(userID int64, purpose string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, t := range m.tokens {
		if t.UserID == userID && t.Purpose == purpose && t.Consumed == nil {
			n++
		}
	}
	return n
}

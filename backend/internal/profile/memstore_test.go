package profile

import (
	"context"
	"sync"
	"time"
)

type memRepo struct {
	mu     sync.Mutex
	byID   map[int64]Record
	byName map[string]int64
}

func newMemRepo() *memRepo {
	return &memRepo{byID: map[int64]Record{}, byName: map[string]int64{}}
}

func (m *memRepo) Upsert(_ context.Context, userID int64, now time.Time) (Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if rec, ok := m.byID[userID]; ok {
		return rec, nil
	}
	rec := Record{UserID: userID, CreatedAt: now, UpdatedAt: now}
	m.byID[userID] = rec
	return rec, nil
}

func (m *memRepo) ByUserID(_ context.Context, userID int64) (Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.byID[userID]
	if !ok {
		return Record{}, ErrNotFound
	}
	return rec, nil
}

func (m *memRepo) ByUsername(_ context.Context, username string) (Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byName[username]
	if !ok {
		return Record{}, ErrNotFound
	}
	return m.byID[id], nil
}

func (m *memRepo) UpdateBio(_ context.Context, userID int64, bio string, now time.Time) (Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.byID[userID]
	if !ok {
		return Record{}, ErrNotFound
	}
	rec.Bio = bio
	rec.UpdatedAt = now
	m.byID[userID] = rec
	return rec, nil
}

func (m *memRepo) UpdateUsername(_ context.Context, userID int64, username string, now time.Time) (Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.byID[userID]
	if !ok {
		return Record{}, ErrNotFound
	}
	if other, taken := m.byName[username]; taken && other != userID {
		return Record{}, ErrUsernameTaken
	}
	if rec.Username != "" {
		delete(m.byName, rec.Username)
	}
	rec.Username = username
	rec.UsernameChangedAt = now
	rec.UpdatedAt = now
	m.byID[userID] = rec
	m.byName[username] = userID
	return rec, nil
}

func (m *memRepo) SetAvatar(_ context.Context, userID int64, key string, now time.Time) (Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.byID[userID]
	if !ok {
		return Record{}, ErrNotFound
	}
	rec.AvatarObjectKey = key
	rec.UpdatedAt = now
	m.byID[userID] = rec
	return rec, nil
}

func (m *memRepo) SetCover(_ context.Context, userID int64, key string, now time.Time) (Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.byID[userID]
	if !ok {
		return Record{}, ErrNotFound
	}
	rec.CoverObjectKey = key
	rec.UpdatedAt = now
	m.byID[userID] = rec
	return rec, nil
}

func (m *memRepo) InsertMedia(context.Context, Media) error { return nil }

type memAccounts struct {
	mu   sync.Mutex
	byID map[int64]Account
}

func newMemAccounts() *memAccounts {
	return &memAccounts{byID: map[int64]Account{}}
}

func (m *memAccounts) put(a Account) { m.mu.Lock(); m.byID[a.ID] = a; m.mu.Unlock() }

func (m *memAccounts) ByID(_ context.Context, id int64) (Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.byID[id]
	if !ok {
		return Account{}, ErrNotFound
	}
	return a, nil
}

func (m *memAccounts) UpdateDisplayName(_ context.Context, id int64, name string, _ time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.byID[id]
	if !ok {
		return ErrNotFound
	}
	a.DisplayName = name
	m.byID[id] = a
	return nil
}

type staticBlocks struct {
	pairs map[[2]int64]struct{}
}

func (s staticBlocks) Blocked(_ context.Context, viewerID, ownerID int64) (bool, error) {
	if s.pairs == nil {
		return false, nil
	}
	_, ok := s.pairs[[2]int64{viewerID, ownerID}]
	return ok, nil
}

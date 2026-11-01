// Package session is the Redis-backed server-side session store.
package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	keyPrefix = "session:"
	idBytes   = 32
	// TTL is the idle/absolute lifetime of a session.
	TTL = 30 * 24 * time.Hour
	// TouchMinAge is how long between sliding TTL refreshes.
	TouchMinAge  = time.Hour
	maxUserAgent = 256
)

// ErrNotFound is a missing or expired session.
var ErrNotFound = errors.New("session: not found")

// Record is the JSON value stored at session:{id}. Keep it well under 512 bytes.
type Record struct {
	UserID    int64     `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
	TouchedAt time.Time `json:"touchedAt"`
	UserAgent string    `json:"userAgent"`
	IP        string    `json:"ip"`
}

// Store persists sessions in Redis.
type Store struct {
	rdb redis.Cmdable
	now func() time.Time
}

// NewStore returns a session store. rdb must not be nil.
func NewStore(rdb redis.Cmdable) *Store {
	return &Store{rdb: rdb, now: time.Now}
}

// Create writes a new session and returns the opaque id (not logged by callers).
func (s *Store) Create(ctx context.Context, rec Record) (string, error) {
	id, err := newID()
	if err != nil {
		return "", err
	}
	now := s.now().UTC()
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = now
	}
	rec.TouchedAt = now
	rec.UserAgent = truncate(rec.UserAgent, maxUserAgent)
	if err := s.set(ctx, id, rec, TTL); err != nil {
		return "", err
	}
	return id, nil
}

// Get loads a session. It does not slide the TTL.
func (s *Store) Get(ctx context.Context, id string) (Record, error) {
	if id == "" {
		return Record{}, ErrNotFound
	}
	raw, err := s.rdb.Get(ctx, keyPrefix+id).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return Record{}, ErrNotFound
		}
		return Record{}, fmt.Errorf("session get: %w", err)
	}
	var rec Record
	if err := json.Unmarshal(raw, &rec); err != nil {
		return Record{}, fmt.Errorf("session decode: %w", err)
	}
	return rec, nil
}

// GetAndTouch loads a session and refreshes TTL at most once per TouchMinAge.
func (s *Store) GetAndTouch(ctx context.Context, id string) (Record, error) {
	rec, err := s.Get(ctx, id)
	if err != nil {
		return Record{}, err
	}
	now := s.now().UTC()
	if now.Sub(rec.TouchedAt) < TouchMinAge {
		return rec, nil
	}
	rec.TouchedAt = now
	if err := s.set(ctx, id, rec, TTL); err != nil {
		return Record{}, err
	}
	return rec, nil
}

// Delete revokes a session. Missing keys are not an error.
func (s *Store) Delete(ctx context.Context, id string) error {
	if id == "" {
		return nil
	}
	if err := s.rdb.Del(ctx, keyPrefix+id).Err(); err != nil {
		return fmt.Errorf("session delete: %w", err)
	}
	return nil
}

func (s *Store) set(ctx context.Context, id string, rec Record, ttl time.Duration) error {
	body, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("session encode: %w", err)
	}
	if err := s.rdb.Set(ctx, keyPrefix+id, body, ttl).Err(); err != nil {
		return fmt.Errorf("session set: %w", err)
	}
	return nil
}

func newID() (string, error) {
	var b [idBytes]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("session id: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

func truncate(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n]
}

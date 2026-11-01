package session

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func testStore(t *testing.T) (*Store, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return NewStore(rdb), mr
}

func TestCreateGetDelete(t *testing.T) {
	t.Parallel()
	s, mr := testStore(t)
	ctx := context.Background()
	id, err := s.Create(ctx, Record{UserID: 99, UserAgent: "vitest", IP: "127.0.0.1"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(id) != 64 {
		t.Fatalf("id length = %d, want 64 hex chars", len(id))
	}
	rec, err := s.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if rec.UserID != 99 {
		t.Errorf("UserID = %d", rec.UserID)
	}
	if rec.UserAgent != "vitest" || rec.IP != "127.0.0.1" {
		t.Errorf("metadata = %+v", rec)
	}
	if rec.CreatedAt.IsZero() || rec.TouchedAt.IsZero() {
		t.Error("timestamps unset")
	}
	ttl := mr.TTL("session:" + id)
	if ttl <= 29*24*time.Hour {
		t.Errorf("ttl = %s, want ~30d", ttl)
	}

	raw, err := mr.Get("session:" + id)
	if err != nil {
		t.Fatalf("raw get: %v", err)
	}
	if strings.Contains(raw, "password") {
		t.Fatalf("session value must not mention password: %s", raw)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("json: %v", err)
	}
	if _, ok := payload["userId"]; !ok {
		t.Errorf("missing userId in %s", raw)
	}

	if err := s.Delete(ctx, id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(ctx, id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get after delete: %v, want ErrNotFound", err)
	}
}

func TestGetMissing(t *testing.T) {
	t.Parallel()
	s, _ := testStore(t)
	if _, err := s.Get(context.Background(), "deadbeef"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestGetAndTouchSlidesOncePerHour(t *testing.T) {
	t.Parallel()
	s, mr := testStore(t)
	ctx := context.Background()
	start := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return start }

	id, err := s.Create(ctx, Record{UserID: 1})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	mr.SetTTL("session:"+id, 10*time.Hour)

	s.now = func() time.Time { return start.Add(30 * time.Minute) }
	if _, err := s.GetAndTouch(ctx, id); err != nil {
		t.Fatalf("touch early: %v", err)
	}
	if mr.TTL("session:"+id) > 10*time.Hour {
		t.Fatal("TTL refreshed before one hour")
	}

	s.now = func() time.Time { return start.Add(TouchMinAge + time.Minute) }
	if _, err := s.GetAndTouch(ctx, id); err != nil {
		t.Fatalf("touch late: %v", err)
	}
	if mr.TTL("session:"+id) < 29*24*time.Hour {
		t.Errorf("TTL after slide = %s, want ~30d", mr.TTL("session:"+id))
	}
}

func TestCreateTruncatesUserAgent(t *testing.T) {
	t.Parallel()
	s, _ := testStore(t)
	id, err := s.Create(context.Background(), Record{
		UserID:    1,
		UserAgent: strings.Repeat("a", 400),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	rec, err := s.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(rec.UserAgent) != maxUserAgent {
		t.Fatalf("UserAgent len = %d, want %d", len(rec.UserAgent), maxUserAgent)
	}
}

package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func testLimiter(t *testing.T) *Limiter {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return New(rdb)
}

func TestAllowWithinLimit(t *testing.T) {
	t.Parallel()
	l := testLimiter(t)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		res, err := l.Allow(ctx, "rate:auth:login:ip:1", 3, time.Minute)
		if err != nil {
			t.Fatalf("Allow %d: %v", i, err)
		}
		if !res.Allowed {
			t.Fatalf("Allow %d denied", i)
		}
	}
	res, err := l.Allow(ctx, "rate:auth:login:ip:1", 3, time.Minute)
	if err != nil {
		t.Fatalf("Allow overflow: %v", err)
	}
	if res.Allowed {
		t.Fatal("4th event should be denied")
	}
	if res.RetryAfter <= 0 {
		t.Errorf("RetryAfter = %s, want positive", res.RetryAfter)
	}
}

func TestAllowIndependentKeys(t *testing.T) {
	t.Parallel()
	l := testLimiter(t)
	ctx := context.Background()
	if _, err := l.Allow(ctx, "rate:auth:login:ip:a", 1, time.Minute); err != nil {
		t.Fatal(err)
	}
	res, err := l.Allow(ctx, "rate:auth:login:ip:b", 1, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Allowed {
		t.Fatal("other key should be allowed")
	}
}

func TestAllowRejectsBadArgs(t *testing.T) {
	t.Parallel()
	l := testLimiter(t)
	ctx := context.Background()
	if _, err := l.Allow(ctx, "k", 0, time.Minute); err == nil {
		t.Fatal("expected error for limit 0")
	}
	if _, err := l.Allow(ctx, "k", 1, 0); err == nil {
		t.Fatal("expected error for window 0")
	}
}

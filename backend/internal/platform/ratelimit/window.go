// Package ratelimit is a Redis sliding-window limiter for the platform kernel.
package ratelimit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Lua: drop expired members, add this event, expire the key, return
// {allowed (1/0), retry_after_seconds}.
var slidingScript = redis.NewScript(`
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window_ms = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local member = ARGV[4]
redis.call('ZREMRANGEBYSCORE', key, 0, now - window_ms)
redis.call('ZADD', key, now, member)
local count = redis.call('ZCARD', key)
redis.call('PEXPIRE', key, window_ms)
if count > limit then
  local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
  local retry = math.ceil(window_ms / 1000)
  if oldest[2] then
    retry = math.ceil((tonumber(oldest[2]) + window_ms - now) / 1000)
    if retry < 1 then
      retry = 1
    end
  end
  return {0, retry}
end
return {1, 0}
`)

// Limiter is a Redis sliding window.
type Limiter struct {
	rdb redis.Cmdable
	now func() time.Time
}

// New returns a limiter. rdb must not be nil.
func New(rdb redis.Cmdable) *Limiter {
	return &Limiter{rdb: rdb, now: time.Now}
}

// Result is one Allow decision.
type Result struct {
	Allowed    bool
	RetryAfter time.Duration
}

// Allow records one event at key and returns whether it is inside limit
// over window. Keys should already include the namespace, e.g.
// rate:auth:login:ip:203.0.113.9
func (l *Limiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (Result, error) {
	if limit <= 0 {
		return Result{}, fmt.Errorf("ratelimit: limit must be positive")
	}
	if window <= 0 {
		return Result{}, fmt.Errorf("ratelimit: window must be positive")
	}
	now := l.now().UTC().UnixMilli()
	member, err := uniqueMember(now)
	if err != nil {
		return Result{}, err
	}
	raw, err := slidingScript.Run(ctx, l.rdb, []string{key}, now, window.Milliseconds(), limit, member).Slice()
	if err != nil {
		return Result{}, fmt.Errorf("ratelimit allow: %w", err)
	}
	if len(raw) != 2 {
		return Result{}, fmt.Errorf("ratelimit allow: unexpected redis reply")
	}
	allowed, err := toInt64(raw[0])
	if err != nil {
		return Result{}, fmt.Errorf("ratelimit allow: %w", err)
	}
	retry, err := toInt64(raw[1])
	if err != nil {
		return Result{}, fmt.Errorf("ratelimit allow: %w", err)
	}
	return Result{
		Allowed:    allowed == 1,
		RetryAfter: time.Duration(retry) * time.Second,
	}, nil
}

func uniqueMember(nowMs int64) (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("ratelimit member: %w", err)
	}
	return fmt.Sprintf("%d-%s", nowMs, hex.EncodeToString(b[:])), nil
}

func toInt64(v any) (int64, error) {
	switch n := v.(type) {
	case int64:
		return n, nil
	case int:
		return int64(n), nil
	default:
		return 0, fmt.Errorf("want int, got %T", v)
	}
}

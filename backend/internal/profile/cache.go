package profile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	pubKeyPrefix = "profile:pub:"
	uidKeyPrefix = "profile:uid:"
	cacheTTL     = 5 * time.Minute
)

type publicCache struct {
	rdb redis.Cmdable
}

func (c *publicCache) get(ctx context.Context, username string) (Public, bool, error) {
	if c == nil || c.rdb == nil || username == "" {
		return Public{}, false, nil
	}
	raw, err := c.rdb.Get(ctx, pubKeyPrefix+username).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return Public{}, false, nil
		}
		return Public{}, false, fmt.Errorf("profile cache get: %w", err)
	}
	var p Public
	if err := json.Unmarshal(raw, &p); err != nil {
		return Public{}, false, fmt.Errorf("profile cache decode: %w", err)
	}
	return p, true, nil
}

func (c *publicCache) set(ctx context.Context, p Public) error {
	if c == nil || c.rdb == nil || p.Username == "" {
		return nil
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("profile cache encode: %w", err)
	}
	pipe := c.rdb.TxPipeline()
	pipe.Set(ctx, pubKeyPrefix+p.Username, raw, cacheTTL)
	pipe.Set(ctx, uidKeyPrefix+strconv.FormatInt(p.ID, 10), p.Username, cacheTTL)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("profile cache set: %w", err)
	}
	return nil
}

func (c *publicCache) invalidate(ctx context.Context, userID int64, usernames ...string) {
	if c == nil || c.rdb == nil {
		return
	}
	keys := make([]string, 0, len(usernames)+2)
	uidKey := uidKeyPrefix + strconv.FormatInt(userID, 10)
	prev, err := c.rdb.Get(ctx, uidKey).Result()
	if err == nil && prev != "" {
		keys = append(keys, pubKeyPrefix+prev)
	}
	for _, u := range usernames {
		if u != "" {
			keys = append(keys, pubKeyPrefix+u)
		}
	}
	keys = append(keys, uidKey)
	_ = c.rdb.Del(ctx, keys...).Err()
}

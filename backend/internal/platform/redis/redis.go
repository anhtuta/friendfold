// Package redisx wraps a Redis client for the platform kernel.
package redisx

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Client is a Redis client whose Ping matches the HTTP ready-check interface.
type Client struct {
	inner *redis.Client
}

// New parses url and returns a client. It does not ping; a down Redis must not
// prevent the process from serving /v1/health.
func New(url string) (*Client, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("redis: parse url: %w", err)
	}
	return &Client{inner: redis.NewClient(opts)}, nil
}

// Ping issues a Redis PING.
func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.inner == nil {
		return fmt.Errorf("redis ping: client is nil")
	}
	if err := c.inner.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}
	return nil
}

// Close closes the underlying client.
func (c *Client) Close() error {
	if c == nil || c.inner == nil {
		return nil
	}
	return c.inner.Close()
}

// Cmdable returns the underlying Redis client for session and rate-limit use.
func (c *Client) Cmdable() redis.Cmdable {
	if c == nil {
		return nil
	}
	return c.inner
}

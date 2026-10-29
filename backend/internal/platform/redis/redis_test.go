package redisx

import (
	"testing"
)

func TestNewRejectsInvalidURL(t *testing.T) {
	t.Parallel()
	if _, err := New("://not-a-url"); err == nil {
		t.Fatal("expected error for invalid Redis URL")
	}
}

func TestNewAcceptsRedisURL(t *testing.T) {
	t.Parallel()
	c, err := New("redis://127.0.0.1:6379/0")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

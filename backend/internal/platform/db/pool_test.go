package db

import (
	"context"
	"testing"

	"github.com/anhtuta/friendfold/backend/internal/platform/config"
)

func TestNewRejectsInvalidWriteURL(t *testing.T) {
	t.Parallel()
	_, err := New(context.Background(), config.Config{
		DatabaseURL:     "://not-a-url",
		DatabaseReadURL: "postgres://friendfold:friendfold@127.0.0.1:5432/friendfold",
		PGXMaxConns:     10,
	})
	if err == nil {
		t.Fatal("expected error for invalid write URL")
	}
}

func TestNewRejectsInvalidReadURL(t *testing.T) {
	t.Parallel()
	_, err := New(context.Background(), config.Config{
		DatabaseURL:     "postgres://friendfold:friendfold@127.0.0.1:5432/friendfold",
		DatabaseReadURL: "://not-a-url",
		PGXMaxConns:     10,
	})
	if err == nil {
		t.Fatal("expected error for invalid read URL")
	}
}

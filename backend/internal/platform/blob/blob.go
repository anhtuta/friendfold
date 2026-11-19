// Package blob is the object-store kernel (ADR 0009). Feature modules persist
// already-trusted bytes; they do not open files or speak S3.
package blob

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// ErrNotFound is a missing object.
var ErrNotFound = errors.New("blob: not found")

// ErrInvalidKey is a key that failed the allowlist (path traversal, wrong shape).
var ErrInvalidKey = errors.New("blob: invalid key")

// Store is durable object storage. Keys are server-minted; callers never take
// a user-supplied name.
type Store interface {
	Put(ctx context.Context, key, contentType string, r io.Reader) error
	Get(ctx context.Context, key string) (io.ReadCloser, string, error)
	Delete(ctx context.Context, key string) error
}

// Driver names for Config.Driver.
const (
	DriverFS = "fs"
	DriverS3 = "s3"
)

// Config selects a driver. PublicBaseURL is not used by the store; compose
// JSON URLs with PublicURL.
type Config struct {
	Driver        string
	Dir           string
	PublicBaseURL string
	S3            S3Config
}

// S3Config is the S3-compatible backend (local RustFS, R2, AWS).
type S3Config struct {
	Endpoint  string
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
	UseSSL    bool
	PathStyle bool
}

// keyRe is the only shape Put/Get/Delete accept. Snowflake object ids are
// decimal; tests and OpenAPI examples may use a hyphenated fake like
// example-object.
var keyRe = regexp.MustCompile(`^profile/[0-9]+/(avatar|cover)/[0-9a-z-]+\.jpg$`)

// ValidateKey rejects path traversal and anything outside the profile key layout.
func ValidateKey(key string) error {
	if key == "" || strings.Contains(key, "..") || strings.HasPrefix(key, "/") {
		return fmt.Errorf("%w", ErrInvalidKey)
	}
	if !keyRe.MatchString(key) {
		return fmt.Errorf("%w", ErrInvalidKey)
	}
	return nil
}

// PublicURL joins MEDIA_PUBLIC_BASE_URL and key. Switching CDN is an env change.
func PublicURL(base, key string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" || key == "" {
		return ""
	}
	return base + "/" + key
}

// Open builds a Store from cfg. ctx is used to create the S3 bucket if needed.
func Open(ctx context.Context, cfg Config) (Store, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Driver)) {
	case "", DriverFS:
		return NewFS(cfg.Dir)
	case DriverS3:
		return NewS3(ctx, cfg.S3)
	default:
		return nil, fmt.Errorf("blob: unknown MEDIA_DRIVER %q (want fs or s3)", cfg.Driver)
	}
}

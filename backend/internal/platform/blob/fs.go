package blob

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const jpegType = "image/jpeg"

// FS stores objects as files under Root.
type FS struct {
	root string
}

// NewFS creates Root if needed and returns a filesystem store.
func NewFS(root string) (*FS, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("blob: MEDIA_DIR is required for fs driver")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("blob: abs MEDIA_DIR: %w", err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, fmt.Errorf("blob: mkdir MEDIA_DIR: %w", err)
	}
	return &FS{root: abs}, nil
}

func (s *FS) resolve(key string) (string, error) {
	if err := ValidateKey(key); err != nil {
		return "", err
	}
	// FromSlash so keys stay forward-slash on all platforms.
	path := filepath.Join(s.root, filepath.FromSlash(key))
	rel, err := filepath.Rel(s.root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w", ErrInvalidKey)
	}
	return path, nil
}

// Put writes key atomically (temp file + rename).
func (s *FS) Put(ctx context.Context, key, contentType string, r io.Reader) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := s.resolve(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("blob: mkdir object dir: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".put-*")
	if err != nil {
		return fmt.Errorf("blob: create temp: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := io.Copy(tmp, r); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("blob: write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("blob: close temp: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("blob: rename: %w", err)
	}
	return nil
}

// Get opens the file. Content type is image/jpeg (all stored objects are JPEG).
func (s *FS) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	path, err := s.resolve(key)
	if err != nil {
		return nil, "", err
	}
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, "", ErrNotFound
		}
		return nil, "", fmt.Errorf("blob: open: %w", err)
	}
	return f, jpegType, nil
}

// Delete removes key. Missing keys are ErrNotFound.
func (s *FS) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := s.resolve(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrNotFound
		}
		return fmt.Errorf("blob: remove: %w", err)
	}
	return nil
}

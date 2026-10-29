package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// argon2id parameters (OWASP 2023 minimums): 19 MiB, time 2, parallelism 1.
const (
	argonMemoryKiB   = 19 * 1024
	argonTime        = 2
	argonThreads     = uint8(1)
	argonSaltLen     = 16
	argonKeyLen      = 32
	maxPasswordBytes = 72
	minPasswordBytes = 8
)

// Hasher hashes and verifies passwords on a bounded worker pool so a login
// flood cannot starve the process.
type Hasher struct {
	sem       chan struct{}
	dummyHash string
}

// NewHasher returns a hasher with at most concurrency in-flight argon2 jobs.
func NewHasher(concurrency int) (*Hasher, error) {
	if concurrency < 1 {
		concurrency = 1
	}
	h := &Hasher{sem: make(chan struct{}, concurrency)}
	dummy, err := h.hashUnlocked("timing-dummy-not-a-user-password")
	if err != nil {
		return nil, err
	}
	h.dummyHash = dummy
	return h, nil
}

// Hash returns a PHC-encoded argon2id string.
func (h *Hasher) Hash(ctx context.Context, password string) (string, error) {
	if err := h.acquire(ctx); err != nil {
		return "", err
	}
	defer h.release()
	return h.hashUnlocked(password)
}

// Compare reports whether password matches encoded. It is constant-time in the
// derived key (argon2 library + subtle).
func (h *Hasher) Compare(ctx context.Context, encoded, password string) (bool, error) {
	if err := h.acquire(ctx); err != nil {
		return false, err
	}
	defer h.release()
	return compareUnlocked(encoded, password)
}

// CompareDummy spends the same argon2 work as a real compare so missing users
// do not return faster than a bad password.
func (h *Hasher) CompareDummy(ctx context.Context, password string) error {
	_, err := h.Compare(ctx, h.dummyHash, password)
	return err
}

func (h *Hasher) acquire(ctx context.Context) error {
	select {
	case h.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("auth hash: %w", ctx.Err())
	}
}

func (h *Hasher) release() {
	<-h.sem
}

func (h *Hasher) hashUnlocked(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("auth hash salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemoryKiB, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemoryKiB, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

func compareUnlocked(encoded, password string) (bool, error) {
	memory, timeCost, threads, salt, want, err := parsePHC(encoded)
	if err != nil {
		return false, err
	}
	got := argon2.IDKey([]byte(password), salt, timeCost, memory, threads, uint32(len(want)))
	if subtle.ConstantTimeCompare(got, want) == 1 {
		return true, nil
	}
	return false, nil
}

func parsePHC(encoded string) (memory, timeCost uint32, threads uint8, salt, key []byte, err error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return 0, 0, 0, nil, nil, fmt.Errorf("auth: invalid password hash")
	}
	var m, t, p int
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil {
		return 0, 0, 0, nil, nil, fmt.Errorf("auth: invalid password hash")
	}
	if m < 1 || t < 1 || p < 1 || p > 255 {
		return 0, 0, 0, nil, nil, fmt.Errorf("auth: invalid password hash")
	}
	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return 0, 0, 0, nil, nil, fmt.Errorf("auth: invalid password hash")
	}
	key, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return 0, 0, 0, nil, nil, fmt.Errorf("auth: invalid password hash")
	}
	return uint32(m), uint32(t), uint8(p), salt, key, nil
}

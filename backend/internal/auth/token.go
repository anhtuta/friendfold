package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

const (
	PurposeVerifyEmail   = "verify_email"
	PurposeResetPassword = "reset_password"
	tokenRawBytes        = 32
	maxTokenChars        = 128
	verifyTTL            = 24 * time.Hour
	resetTTL             = time.Hour
)

// NewVerifyToken returns a raw URL token and its SHA-256 hash. Only the hash
// is stored. The raw value is ≥ 32 bytes of randomness, unpadded base64url.
func NewVerifyToken() (raw string, hash []byte, err error) {
	b := make([]byte, tokenRawBytes)
	if _, err := rand.Read(b); err != nil {
		return "", nil, fmt.Errorf("auth token: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(raw))
	return raw, sum[:], nil
}

// HashToken SHA-256s the presented secret. Empty is a validation error; longer
// than 128 characters is invalid_token without a lookup.
func HashToken(raw string) ([]byte, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil, ValidationError{Msg: "token is required"}
	}
	if len(s) > maxTokenChars {
		return nil, ErrInvalidToken
	}
	sum := sha256.Sum256([]byte(s))
	return sum[:], nil
}

package auth

import (
	"encoding/base64"
	"errors"
	"testing"
)

func TestNewVerifyTokenHashedNotEqualRaw(t *testing.T) {
	t.Parallel()
	raw, hash, err := NewVerifyToken()
	if err != nil {
		t.Fatalf("NewVerifyToken: %v", err)
	}
	if len(raw) < 43 {
		t.Fatalf("raw too short: %q", raw)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(decoded) < tokenRawBytes {
		t.Fatalf("decoded %d bytes, want ≥ %d", len(decoded), tokenRawBytes)
	}
	if string(hash) == raw {
		t.Fatal("hash equals raw token")
	}
	again, err := HashToken(raw)
	if err != nil {
		t.Fatalf("HashToken: %v", err)
	}
	if string(again) != string(hash) {
		t.Fatal("HashToken does not match NewVerifyToken hash")
	}
	other, _, err := NewVerifyToken()
	if err != nil {
		t.Fatal(err)
	}
	if other == raw {
		t.Fatal("tokens not unique")
	}
}

func TestHashTokenEmptyAndTooLong(t *testing.T) {
	t.Parallel()
	_, err := HashToken("  ")
	var ve ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("empty: %v", err)
	}
	_, err = HashToken(string(make([]byte, maxTokenChars+1)))
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("too long: %v", err)
	}
}

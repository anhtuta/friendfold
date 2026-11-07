package auth

import (
	"context"
	"strings"
	"testing"
)

func TestHashAndCompare(t *testing.T) {
	t.Parallel()
	h, err := NewHasher(1)
	if err != nil {
		t.Fatalf("NewHasher: %v", err)
	}
	ctx := context.Background()
	encoded, err := h.Hash(ctx, "correct-horse-battery")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if !strings.HasPrefix(encoded, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("unexpected encoding %q", encoded)
	}
	ok, err := h.Compare(ctx, encoded, "correct-horse-battery")
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	if !ok {
		t.Fatal("expected match")
	}
	ok, err = h.Compare(ctx, encoded, "wrong-password-ok")
	if err != nil {
		t.Fatalf("Compare wrong: %v", err)
	}
	if ok {
		t.Fatal("wrong password matched")
	}
}

func TestHashIsNotPlaintext(t *testing.T) {
	t.Parallel()
	h, err := NewHasher(1)
	if err != nil {
		t.Fatalf("NewHasher: %v", err)
	}
	pw := "correct-horse-battery"
	encoded, err := h.Hash(context.Background(), pw)
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if strings.Contains(encoded, pw) {
		t.Fatal("hash contains plaintext password")
	}
}

func TestValidatePasswordBounds(t *testing.T) {
	t.Parallel()
	if err := validatePassword("short", "a@b.c"); err == nil {
		t.Fatal("expected error for short password")
	}
	if err := validatePassword(strings.Repeat("a", 73), "a@b.c"); err == nil {
		t.Fatal("expected error for 73-byte password")
	}
	if err := validatePassword("Minh@example.com", "minh@example.com"); err == nil {
		t.Fatal("expected error when password equals email")
	}
	if err := validatePassword("correct-horse", "minh@example.com"); err != nil {
		t.Fatalf("valid password rejected: %v", err)
	}
}

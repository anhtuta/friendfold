package profile

import (
	"errors"
	"testing"
)

func TestNormalizeUsername(t *testing.T) {
	t.Parallel()
	got, err := normalizeUsername("Minh_1")
	if err != nil || got != "minh_1" {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := normalizeUsername("ab"); !errors.Is(err, ErrUsernameInvalid) {
		t.Errorf("short: %v", err)
	}
	if _, err := normalizeUsername("Bad Name"); !errors.Is(err, ErrUsernameInvalid) {
		t.Errorf("space: %v", err)
	}
	if _, err := normalizeUsername("admin"); !errors.Is(err, ErrUsernameReserved) {
		t.Errorf("reserved: %v", err)
	}
	if _, err := normalizeUsername("Friendfold"); !errors.Is(err, ErrUsernameReserved) {
		t.Errorf("reserved case: %v", err)
	}
}

func TestNormalizeBioAndDisplayName(t *testing.T) {
	t.Parallel()
	if _, err := normalizeDisplayName("  Minh  "); err != nil {
		t.Fatal(err)
	}
	if _, err := normalizeDisplayName(""); err == nil {
		t.Fatal("empty display name")
	}
	if _, err := normalizeDisplayName("a\nb"); err == nil {
		t.Fatal("newline display name")
	}
	if _, err := normalizeBio(""); err != nil {
		t.Fatal(err)
	}
	long := make([]rune, 161)
	for i := range long {
		long[i] = 'a'
	}
	if _, err := normalizeBio(string(long)); err == nil {
		t.Fatal("bio too long")
	}
}

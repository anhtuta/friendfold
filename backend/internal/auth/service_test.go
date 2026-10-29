package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/anhtuta/friendfold/backend/internal/platform/id"
	"github.com/anhtuta/friendfold/backend/internal/platform/ratelimit"
	"github.com/anhtuta/friendfold/backend/internal/platform/session"
)

func testService(t *testing.T) (*Service, *memStore) {
	t.Helper()
	h, err := NewHasher(1)
	if err != nil {
		t.Fatalf("hasher: %v", err)
	}
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	gen, err := id.NewGenerator(1)
	if err != nil {
		t.Fatalf("id: %v", err)
	}
	users := newMemStore()
	svc := NewService(users, session.NewStore(rdb), h, gen, ratelimit.New(rdb), nil)
	svc.now = func() time.Time {
		return time.Date(2026, 10, 29, 15, 0, 0, 0, time.UTC)
	}
	return svc, users
}

func TestAgeGate(t *testing.T) {
	t.Parallel()
	today := time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC)
	if err := checkAge(time.Date(2013, 10, 30, 0, 0, 0, 0, time.UTC), today); !errors.Is(err, ErrUnderage) {
		t.Fatalf("12 years old yesterday: %v", err)
	}
	if err := checkAge(time.Date(2013, 10, 29, 0, 0, 0, 0, time.UTC), today); err != nil {
		t.Fatalf("turns 13 today: %v", err)
	}
	if err := checkAge(time.Date(2013, 10, 28, 0, 0, 0, 0, time.UTC), today); err != nil {
		t.Fatalf("already 13: %v", err)
	}
}

func TestRegisterAndLogin(t *testing.T) {
	t.Parallel()
	svc, users := testService(t)
	ctx := context.Background()
	in := RegisterInput{
		Email:         "Minh@Example.com",
		Password:      "correct-horse-battery",
		DisplayName:   "Minh",
		DateOfBirth:   time.Date(1998, 4, 12, 0, 0, 0, 0, time.UTC),
		TermsAccepted: true,
		IP:            "127.0.0.1",
	}
	out, err := svc.Register(ctx, in)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if out.User.Email != "minh@example.com" {
		t.Errorf("email = %q", out.User.Email)
	}
	if out.User.EmailVerified {
		t.Error("emailVerified should be false in phase 1")
	}
	if out.SessionID == "" {
		t.Fatal("missing session id")
	}
	stored, err := users.ByEmail(ctx, "minh@example.com")
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	if stored.PasswordHash == "" || stored.PasswordHash == in.Password {
		t.Fatal("password stored plaintext or empty")
	}
	if !stringsHasPrefix(stored.PasswordHash, "$argon2id$") {
		t.Fatalf("hash = %q", stored.PasswordHash)
	}

	_, err = svc.Register(ctx, in)
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("duplicate: %v", err)
	}

	login, err := svc.Login(ctx, LoginInput{Email: "minh@example.com", Password: in.Password, IP: "127.0.0.1"})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	me, err := svc.Me(ctx, login.SessionID)
	if err != nil {
		t.Fatalf("Me: %v", err)
	}
	if me.ID != out.User.ID {
		t.Errorf("me id = %d", me.ID)
	}
	if err := svc.Logout(ctx, login.SessionID); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if _, err := svc.Me(ctx, login.SessionID); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("Me after logout: %v", err)
	}
}

func TestLoginEnumeration(t *testing.T) {
	t.Parallel()
	svc, _ := testService(t)
	ctx := context.Background()
	_, err := svc.Register(ctx, RegisterInput{
		Email:         "minh@example.com",
		Password:      "correct-horse-battery",
		DisplayName:   "Minh",
		DateOfBirth:   time.Date(1998, 4, 12, 0, 0, 0, 0, time.UTC),
		TermsAccepted: true,
		IP:            "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	unknown, errUnknown := svc.Login(ctx, LoginInput{Email: "nobody@example.com", Password: "correct-horse-battery", IP: "10.0.0.2"})
	if !errors.Is(errUnknown, ErrInvalidCredentials) || unknown.SessionID != "" {
		t.Fatalf("unknown email: user=%v err=%v", unknown, errUnknown)
	}
	bad, errBad := svc.Login(ctx, LoginInput{Email: "minh@example.com", Password: "definitely-wrong-1", IP: "10.0.0.3"})
	if !errors.Is(errBad, ErrInvalidCredentials) || bad.SessionID != "" {
		t.Fatalf("bad password: user=%v err=%v", bad, errBad)
	}
}

func TestRegisterRejectsUnderageAndTerms(t *testing.T) {
	t.Parallel()
	svc, _ := testService(t)
	ctx := context.Background()
	base := RegisterInput{
		Email:       "kid@example.com",
		Password:    "correct-horse-battery",
		DisplayName: "Kid",
		DateOfBirth: time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC),
		IP:          "127.0.0.1",
	}
	base.TermsAccepted = true
	if _, err := svc.Register(ctx, base); !errors.Is(err, ErrUnderage) {
		t.Fatalf("underage: %v", err)
	}
	base.DateOfBirth = time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	base.TermsAccepted = false
	if _, err := svc.Register(ctx, base); !errors.Is(err, ErrTermsRequired) {
		t.Fatalf("terms: %v", err)
	}
}

func stringsHasPrefix(s, p string) bool {
	return len(s) >= len(p) && s[:len(p)] == p
}

package profile

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/anhtuta/friendfold/backend/internal/platform/ratelimit"
)

const testPublicBase = "http://localhost:8080/v1/media"

func testProfileService(t *testing.T) (*Service, *memRepo, *memAccounts, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	repo := newMemRepo()
	acct := newMemAccounts()
	svc := NewService(repo, acct, NoopBlocks{}, rdb, ratelimit.New(rdb), testPublicBase, nil)
	svc.now = func() time.Time {
		return time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	}
	return svc, repo, acct, mr
}

func TestGetMeLazyCreate(t *testing.T) {
	t.Parallel()
	svc, _, acct, _ := testProfileService(t)
	acct.put(Account{ID: 1, Email: "minh@example.com", DisplayName: "Minh"})
	me, err := svc.GetMe(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if me.Email != "minh@example.com" || me.Username != "" || me.DisplayName != "Minh" {
		t.Fatalf("%+v", me)
	}
}

func TestPatchUsernameAndPublicHidesEmail(t *testing.T) {
	t.Parallel()
	svc, _, acct, _ := testProfileService(t)
	ctx := context.Background()
	acct.put(Account{ID: 1, Email: "minh@example.com", EmailVerified: true, DisplayName: "Minh"})
	name := "minh"
	bio := "Into the fold of friends."
	me, err := svc.PatchMe(ctx, PatchInput{UserID: 1, IP: "203.0.113.9", Username: &name, Bio: &bio})
	if err != nil {
		t.Fatal(err)
	}
	if me.Username != "minh" || me.Email != "minh@example.com" {
		t.Fatalf("%+v", me)
	}
	pub, err := svc.GetPublic(ctx, "Minh", 0, "203.0.113.9")
	if err != nil {
		t.Fatal(err)
	}
	if pub.Username != "minh" || pub.Bio != bio || pub.DisplayName != "Minh" {
		t.Fatalf("%+v", pub)
	}
	raw, _ := json.Marshal(pub)
	if strings.Contains(string(raw), "email") || strings.Contains(string(raw), "minh@example.com") {
		t.Fatalf("public leaked email: %s", raw)
	}
}

func TestReservedAndTakenUsername(t *testing.T) {
	t.Parallel()
	svc, _, acct, _ := testProfileService(t)
	ctx := context.Background()
	acct.put(Account{ID: 1, Email: "a@example.com", DisplayName: "A"})
	acct.put(Account{ID: 2, Email: "b@example.com", DisplayName: "B"})
	admin := "admin"
	if _, err := svc.PatchMe(ctx, PatchInput{UserID: 1, Username: &admin}); !errors.Is(err, ErrUsernameReserved) {
		t.Fatalf("reserved: %v", err)
	}
	u := "minh"
	if _, err := svc.PatchMe(ctx, PatchInput{UserID: 1, Username: &u}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PatchMe(ctx, PatchInput{UserID: 2, Username: &u}); !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("taken: %v", err)
	}
}

func TestUsernameCooldown(t *testing.T) {
	t.Parallel()
	svc, _, acct, _ := testProfileService(t)
	ctx := context.Background()
	acct.put(Account{ID: 1, Email: "a@example.com", DisplayName: "A"})
	first := "minh"
	if _, err := svc.PatchMe(ctx, PatchInput{UserID: 1, Username: &first}); err != nil {
		t.Fatal(err)
	}
	second := "minh2"
	if _, err := svc.PatchMe(ctx, PatchInput{UserID: 1, Username: &second}); !errors.Is(err, ErrUsernameCooldown) {
		t.Fatalf("cooldown: %v", err)
	}
	if _, err := svc.PatchMe(ctx, PatchInput{UserID: 1, Username: &first}); err != nil {
		t.Fatalf("same username should no-op: %v", err)
	}
	svc.now = func() time.Time {
		return time.Date(2026, 12, 2, 12, 0, 0, 0, time.UTC)
	}
	if _, err := svc.PatchMe(ctx, PatchInput{UserID: 1, Username: &second}); err != nil {
		t.Fatalf("after 14d: %v", err)
	}
}

func TestCacheHitMatchesMiss(t *testing.T) {
	t.Parallel()
	svc, _, acct, _ := testProfileService(t)
	ctx := context.Background()
	acct.put(Account{ID: 1, Email: "minh@example.com", DisplayName: "Minh"})
	u := "minh"
	if _, err := svc.PatchMe(ctx, PatchInput{UserID: 1, Username: &u}); err != nil {
		t.Fatal(err)
	}
	miss, err := svc.GetPublic(ctx, "minh", 0, "203.0.113.1")
	if err != nil {
		t.Fatal(err)
	}
	hit, err := svc.GetPublic(ctx, "minh", 0, "203.0.113.1")
	if err != nil {
		t.Fatal(err)
	}
	if miss != hit {
		t.Fatalf("miss %+v hit %+v", miss, hit)
	}
}

func TestBlockedViewerGetsNotFound(t *testing.T) {
	t.Parallel()
	svc, _, acct, _ := testProfileService(t)
	svc.blocks = staticBlocks{pairs: map[[2]int64]struct{}{{2, 1}: {}}}
	ctx := context.Background()
	acct.put(Account{ID: 1, Email: "a@example.com", DisplayName: "A"})
	acct.put(Account{ID: 2, Email: "b@example.com", DisplayName: "B"})
	u := "minh"
	if _, err := svc.PatchMe(ctx, PatchInput{UserID: 1, Username: &u}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetPublic(ctx, "minh", 2, "203.0.113.2"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("blocked: %v", err)
	}
	if _, err := svc.GetPublic(ctx, "minh", 0, "203.0.113.2"); err != nil {
		t.Fatalf("anon: %v", err)
	}
}

func TestPatchDisplayName(t *testing.T) {
	t.Parallel()
	svc, _, acct, _ := testProfileService(t)
	acct.put(Account{ID: 1, Email: "a@example.com", DisplayName: "A"})
	n := "Minh Tran"
	me, err := svc.PatchMe(context.Background(), PatchInput{UserID: 1, DisplayName: &n})
	if err != nil {
		t.Fatal(err)
	}
	if me.DisplayName != "Minh Tran" {
		t.Fatalf("%+v", me)
	}
	got, _ := acct.ByID(context.Background(), 1)
	if got.DisplayName != "Minh Tran" {
		t.Fatalf("account not updated: %+v", got)
	}
}

func TestEmptyPatch(t *testing.T) {
	t.Parallel()
	svc, _, acct, _ := testProfileService(t)
	acct.put(Account{ID: 1, Email: "a@example.com", DisplayName: "A"})
	if _, err := svc.PatchMe(context.Background(), PatchInput{UserID: 1}); !errors.Is(err, ErrValidation) {
		t.Fatalf("%v", err)
	}
}

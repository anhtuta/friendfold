package auth

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/anhtuta/friendfold/backend/internal/platform/id"
	"github.com/anhtuta/friendfold/backend/internal/platform/mail"
	"github.com/anhtuta/friendfold/backend/internal/platform/ratelimit"
	"github.com/anhtuta/friendfold/backend/internal/platform/session"
)

type recordingMail struct {
	mu   sync.Mutex
	msgs []mail.Message
}

func (r *recordingMail) Enqueue(msg mail.Message) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.msgs = append(r.msgs, msg)
}

func (r *recordingMail) last() mail.Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.msgs) == 0 {
		return mail.Message{}
	}
	return r.msgs[len(r.msgs)-1]
}

func (r *recordingMail) len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.msgs)
}

var verifyTokenRE = regexp.MustCompile(`token=([A-Za-z0-9_-]+)`)

func tokenFromMail(t *testing.T, msg mail.Message) string {
	t.Helper()
	m := verifyTokenRE.FindStringSubmatch(msg.Body)
	if len(m) != 2 {
		t.Fatalf("no token in mail body")
	}
	return m[1]
}

func testService(t *testing.T) (*Service, *memStore, *recordingMail) {
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
	rec := &recordingMail{}
	svc := NewService(users, session.NewStore(rdb), h, gen, ratelimit.New(rdb), rec, testOrigin, nil)
	svc.now = func() time.Time {
		return time.Date(2026, 10, 29, 15, 0, 0, 0, time.UTC)
	}
	return svc, users, rec
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
	svc, users, rec := testService(t)
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
		t.Error("emailVerified should be false until the token is consumed")
	}
	if rec.len() != 1 {
		t.Fatalf("verification mail count = %d", rec.len())
	}
	raw := tokenFromMail(t, rec.last())
	if rec.last().To != "minh@example.com" {
		t.Fatalf("mail to = %q", rec.last().To)
	}
	if !strings.Contains(rec.last().Body, "/verify-email?token="+raw) {
		t.Fatal("mail missing verify link")
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
	svc, _, _ := testService(t)
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
	svc, _, _ := testService(t)
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

func TestVerifyEmailOnceThenRejects(t *testing.T) {
	t.Parallel()
	svc, users, rec := testService(t)
	ctx := context.Background()
	out, err := svc.Register(ctx, validRegister("minh@example.com"))
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	raw := tokenFromMail(t, rec.last())
	if err := svc.VerifyEmail(ctx, raw, "10.0.0.8"); err != nil {
		t.Fatalf("VerifyEmail: %v", err)
	}
	me, err := svc.Me(ctx, out.SessionID)
	if err != nil {
		t.Fatalf("Me: %v", err)
	}
	if !me.EmailVerified {
		t.Fatal("expected verified")
	}
	stored, err := users.ByID(ctx, me.ID)
	if err != nil || !stored.EmailVerified {
		t.Fatalf("store verified: %v %+v", err, stored)
	}
	if err := svc.VerifyEmail(ctx, raw, "10.0.0.8"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("second use: %v", err)
	}
}

func TestVerifyEmailExpired(t *testing.T) {
	t.Parallel()
	svc, _, rec := testService(t)
	ctx := context.Background()
	start := time.Date(2026, 10, 29, 15, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return start }
	if _, err := svc.Register(ctx, validRegister("minh@example.com")); err != nil {
		t.Fatalf("Register: %v", err)
	}
	raw := tokenFromMail(t, rec.last())
	svc.now = func() time.Time { return start.Add(25 * time.Hour) }
	if err := svc.VerifyEmail(ctx, raw, "10.0.0.8"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expired: %v", err)
	}
}

func TestResendInvalidatesPreviousToken(t *testing.T) {
	t.Parallel()
	svc, users, rec := testService(t)
	ctx := context.Background()
	out, err := svc.Register(ctx, validRegister("minh@example.com"))
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	first := tokenFromMail(t, rec.last())
	if err := svc.ResendVerification(ctx, out.SessionID, "10.0.0.9"); err != nil {
		t.Fatalf("Resend: %v", err)
	}
	if rec.len() != 2 {
		t.Fatalf("mail count = %d", rec.len())
	}
	second := tokenFromMail(t, rec.last())
	if first == second {
		t.Fatal("resend reused token")
	}
	if users.unusedCount(out.User.ID, PurposeVerifyEmail) != 1 {
		t.Fatalf("unused tokens = %d", users.unusedCount(out.User.ID, PurposeVerifyEmail))
	}
	if err := svc.VerifyEmail(ctx, first, "10.0.0.8"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("old token: %v", err)
	}
	if err := svc.VerifyEmail(ctx, second, "10.0.0.8"); err != nil {
		t.Fatalf("new token: %v", err)
	}
}

func TestResendAlreadyVerifiedNoMail(t *testing.T) {
	t.Parallel()
	svc, _, rec := testService(t)
	ctx := context.Background()
	out, err := svc.Register(ctx, validRegister("minh@example.com"))
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := svc.VerifyEmail(ctx, tokenFromMail(t, rec.last()), "10.0.0.8"); err != nil {
		t.Fatalf("VerifyEmail: %v", err)
	}
	n := rec.len()
	if err := svc.ResendVerification(ctx, out.SessionID, "10.0.0.9"); err != nil {
		t.Fatalf("Resend: %v", err)
	}
	if rec.len() != n {
		t.Fatalf("sent mail after verified: %d vs %d", rec.len(), n)
	}
}

func TestRegisterSucceedsWhenSMTPFails(t *testing.T) {
	t.Parallel()
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
	q := mail.NewQueue(failingSender{}, slog.Default(), 1, context.Background())
	svc := NewService(users, session.NewStore(rdb), h, gen, ratelimit.New(rdb), q, testOrigin, nil)
	svc.now = func() time.Time {
		return time.Date(2026, 10, 29, 15, 0, 0, 0, time.UTC)
	}
	out, err := svc.Register(context.Background(), validRegister("minh@example.com"))
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if out.SessionID == "" || out.User.ID == 0 {
		t.Fatal("signup should succeed without SMTP")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := q.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	if _, err := users.ByEmail(context.Background(), "minh@example.com"); err != nil {
		t.Fatalf("user missing after smtp fail: %v", err)
	}
}

func TestVerifyEmailEmptyToken(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)
	err := svc.VerifyEmail(context.Background(), "  ", "10.0.0.8")
	var ve ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("empty token: %v", err)
	}
}

func TestResendRateLimit(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)
	ctx := context.Background()
	out, err := svc.Register(ctx, validRegister("minh@example.com"))
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	for i := 0; i < resendLimit; i++ {
		if err := svc.ResendVerification(ctx, out.SessionID, "10.0.0.9"); err != nil {
			t.Fatalf("resend %d: %v", i, err)
		}
	}
	err = svc.ResendVerification(ctx, out.SessionID, "10.0.0.9")
	var rl RateLimitError
	if !errors.As(err, &rl) {
		t.Fatalf("expected rate limit, got %v", err)
	}
}

func TestForgotAlways202SendsOnlyIfVerified(t *testing.T) {
	t.Parallel()
	svc, _, rec := testService(t)
	ctx := context.Background()
	if err := svc.ForgotPassword(ctx, "nobody@example.com", "10.0.0.4"); err != nil {
		t.Fatalf("unknown: %v", err)
	}
	if rec.len() != 0 {
		t.Fatal("mailed unknown address")
	}
	out, err := svc.Register(ctx, validRegister("minh@example.com"))
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if rec.len() != 1 {
		t.Fatalf("verify mail = %d", rec.len())
	}
	if err := svc.ForgotPassword(ctx, "minh@example.com", "10.0.0.4"); err != nil {
		t.Fatalf("unverified forgot: %v", err)
	}
	if rec.len() != 1 {
		t.Fatal("mailed unverified address")
	}
	if err := svc.VerifyEmail(ctx, tokenFromMail(t, rec.last()), "10.0.0.8"); err != nil {
		t.Fatalf("verify: %v", err)
	}
	_ = out
	if err := svc.ForgotPassword(ctx, "Minh@Example.com", "10.0.0.4"); err != nil {
		t.Fatalf("verified forgot: %v", err)
	}
	if rec.len() != 2 {
		t.Fatalf("reset mail = %d", rec.len())
	}
	if rec.last().Subject != "Reset your Friendfold password" {
		t.Fatalf("subject = %q", rec.last().Subject)
	}
	raw := tokenFromMail(t, rec.last())
	if !strings.Contains(rec.last().Body, "/reset-password?token="+raw) {
		t.Fatal("mail missing reset link")
	}
}

func TestForgotInvalidatesPreviousResetToken(t *testing.T) {
	t.Parallel()
	svc, users, rec := testService(t)
	ctx := context.Background()
	out, err := svc.Register(ctx, validRegister("minh@example.com"))
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := svc.VerifyEmail(ctx, tokenFromMail(t, rec.last()), "10.0.0.8"); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if err := svc.ForgotPassword(ctx, "minh@example.com", "10.0.0.4"); err != nil {
		t.Fatalf("forgot: %v", err)
	}
	first := tokenFromMail(t, rec.last())
	if err := svc.ForgotPassword(ctx, "minh@example.com", "10.0.0.5"); err != nil {
		t.Fatalf("forgot 2: %v", err)
	}
	second := tokenFromMail(t, rec.last())
	if first == second {
		t.Fatal("reused reset token")
	}
	if users.unusedCount(out.User.ID, PurposeResetPassword) != 1 {
		t.Fatalf("unused reset tokens = %d", users.unusedCount(out.User.ID, PurposeResetPassword))
	}
	if err := svc.ResetPassword(ctx, first, "new-horse-battery-1", "10.0.0.6"); !errors.Is(err, ErrInvalidResetToken) {
		t.Fatalf("old token: %v", err)
	}
	if err := svc.ResetPassword(ctx, second, "new-horse-battery-1", "10.0.0.6"); err != nil {
		t.Fatalf("new token: %v", err)
	}
}

func TestResetPasswordRevokesAllSessionsAndOldPasswordFails(t *testing.T) {
	t.Parallel()
	svc, _, rec := testService(t)
	ctx := context.Background()
	first, err := svc.Register(ctx, validRegister("minh@example.com"))
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := svc.VerifyEmail(ctx, tokenFromMail(t, rec.last()), "10.0.0.8"); err != nil {
		t.Fatalf("verify: %v", err)
	}
	second, err := svc.Login(ctx, LoginInput{Email: "minh@example.com", Password: "correct-horse-battery", IP: "10.0.0.2"})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if err := svc.ForgotPassword(ctx, "minh@example.com", "10.0.0.4"); err != nil {
		t.Fatalf("forgot: %v", err)
	}
	raw := tokenFromMail(t, rec.last())
	if err := svc.ResetPassword(ctx, raw, "new-horse-battery-1", "10.0.0.6"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if _, err := svc.Me(ctx, first.SessionID); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("first session: %v", err)
	}
	if _, err := svc.Me(ctx, second.SessionID); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("second session: %v", err)
	}
	if _, err := svc.Login(ctx, LoginInput{Email: "minh@example.com", Password: "correct-horse-battery", IP: "10.0.0.2"}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("old password: %v", err)
	}
	login, err := svc.Login(ctx, LoginInput{Email: "minh@example.com", Password: "new-horse-battery-1", IP: "10.0.0.2"})
	if err != nil {
		t.Fatalf("new password: %v", err)
	}
	if login.SessionID == first.SessionID || login.SessionID == second.SessionID {
		t.Fatal("reset reused a revoked session id")
	}
	if err := svc.ResetPassword(ctx, raw, "other-horse-battery-2", "10.0.0.6"); !errors.Is(err, ErrInvalidResetToken) {
		t.Fatalf("second use: %v", err)
	}
}

func TestResetRejectsExpiredWrongPurposeAndEmpty(t *testing.T) {
	t.Parallel()
	svc, _, rec := testService(t)
	ctx := context.Background()
	start := time.Date(2026, 10, 29, 15, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return start }
	if _, err := svc.Register(ctx, validRegister("minh@example.com")); err != nil {
		t.Fatalf("Register: %v", err)
	}
	verifyTok := tokenFromMail(t, rec.last())
	if err := svc.VerifyEmail(ctx, verifyTok, "10.0.0.8"); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if err := svc.ForgotPassword(ctx, "minh@example.com", "10.0.0.4"); err != nil {
		t.Fatalf("forgot: %v", err)
	}
	resetTok := tokenFromMail(t, rec.last())
	if err := svc.ResetPassword(ctx, verifyTok, "new-horse-battery-1", "10.0.0.6"); !errors.Is(err, ErrInvalidResetToken) {
		t.Fatalf("verify purpose: %v", err)
	}
	svc.now = func() time.Time { return start.Add(2 * time.Hour) }
	if err := svc.ResetPassword(ctx, resetTok, "new-horse-battery-1", "10.0.0.6"); !errors.Is(err, ErrInvalidResetToken) {
		t.Fatalf("expired: %v", err)
	}
	var ve ValidationError
	if err := svc.ResetPassword(ctx, "  ", "new-horse-battery-1", "10.0.0.6"); !errors.As(err, &ve) {
		t.Fatalf("empty: %v", err)
	}
}

func TestForgotRateLimit(t *testing.T) {
	t.Parallel()
	svc, _, _ := testService(t)
	ctx := context.Background()
	for i := 0; i < forgotEmailLim; i++ {
		if err := svc.ForgotPassword(ctx, "minh@example.com", "10.0.0.4"); err != nil {
			t.Fatalf("forgot %d: %v", i, err)
		}
	}
	err := svc.ForgotPassword(ctx, "minh@example.com", "10.0.0.4")
	var rl RateLimitError
	if !errors.As(err, &rl) {
		t.Fatalf("expected rate limit, got %v", err)
	}
}

func TestChangePasswordRequiresCurrentAndRevokesOthers(t *testing.T) {
	t.Parallel()
	svc, _, rec := testService(t)
	ctx := context.Background()
	first, err := svc.Register(ctx, validRegister("minh@example.com"))
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := svc.VerifyEmail(ctx, tokenFromMail(t, rec.last()), "10.0.0.8"); err != nil {
		t.Fatalf("verify: %v", err)
	}
	second, err := svc.Login(ctx, LoginInput{Email: "minh@example.com", Password: "correct-horse-battery", IP: "10.0.0.2"})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	var ve ValidationError
	if _, err := svc.ChangePassword(ctx, first.SessionID, "", "new-horse-battery-1", "10.0.0.3", "test"); !errors.As(err, &ve) {
		t.Fatalf("missing current: %v", err)
	}
	if _, err := svc.ChangePassword(ctx, first.SessionID, "wrong-horse-battery", "new-horse-battery-1", "10.0.0.3", "test"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong current: %v", err)
	}
	sid, err := svc.ChangePassword(ctx, first.SessionID, "correct-horse-battery", "new-horse-battery-1", "10.0.0.3", "test")
	if err != nil {
		t.Fatalf("change: %v", err)
	}
	if sid == "" || sid == first.SessionID {
		t.Fatal("change should mint a new session")
	}
	if _, err := svc.Me(ctx, first.SessionID); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("old this-device session: %v", err)
	}
	if _, err := svc.Me(ctx, second.SessionID); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("other session: %v", err)
	}
	me, err := svc.Me(ctx, sid)
	if err != nil || me.Email != "minh@example.com" {
		t.Fatalf("new session: %v %+v", err, me)
	}
	if _, err := svc.Login(ctx, LoginInput{Email: "minh@example.com", Password: "correct-horse-battery", IP: "10.0.0.2"}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("old password after change: %v", err)
	}
}

func TestChangeAddPasswordWhenHashEmpty(t *testing.T) {
	t.Parallel()
	svc, users, _ := testService(t)
	ctx := context.Background()
	out, err := svc.Register(ctx, validRegister("minh@example.com"))
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := users.UpdatePasswordHash(ctx, out.User.ID, "", svc.now()); err != nil {
		t.Fatalf("clear hash: %v", err)
	}
	var ve ValidationError
	if _, err := svc.ChangePassword(ctx, out.SessionID, "something", "new-horse-battery-1", "10.0.0.3", "test"); !errors.As(err, &ve) {
		t.Fatalf("current set while adding: %v", err)
	}
	sid, err := svc.ChangePassword(ctx, out.SessionID, "", "new-horse-battery-1", "10.0.0.3", "test")
	if err != nil {
		t.Fatalf("add password: %v", err)
	}
	if _, err := svc.Me(ctx, out.SessionID); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("old session: %v", err)
	}
	if _, err := svc.Login(ctx, LoginInput{Email: "minh@example.com", Password: "new-horse-battery-1", IP: "10.0.0.2"}); err != nil {
		t.Fatalf("login with added password: %v", err)
	}
	_ = sid
}

func validRegister(email string) RegisterInput {
	return RegisterInput{
		Email:         email,
		Password:      "correct-horse-battery",
		DisplayName:   "Minh",
		DateOfBirth:   time.Date(1998, 4, 12, 0, 0, 0, 0, time.UTC),
		TermsAccepted: true,
		IP:            "127.0.0.1",
	}
}

type failingSender struct{}

func (failingSender) Send(context.Context, mail.Message) error {
	return errors.New("smtp unavailable")
}

func stringsHasPrefix(s, p string) bool {
	return len(s) >= len(p) && s[:len(p)] == p
}

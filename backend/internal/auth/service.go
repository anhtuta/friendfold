package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/anhtuta/friendfold/backend/internal/platform/id"
	"github.com/anhtuta/friendfold/backend/internal/platform/mail"
	"github.com/anhtuta/friendfold/backend/internal/platform/ratelimit"
	"github.com/anhtuta/friendfold/backend/internal/platform/session"
)

const (
	minAgeYears      = 13
	maxDisplayRunes  = 80
	registerIPLimit  = 5
	registerEmailLim = 3
	loginLimit       = 10
	verifyIPLimit    = 20
	resendLimit      = 3
	authWindow       = 15 * time.Minute
)

type limiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (ratelimit.Result, error)
}

type sessions interface {
	Create(ctx context.Context, rec session.Record) (string, error)
	GetAndTouch(ctx context.Context, id string) (session.Record, error)
	Delete(ctx context.Context, id string) error
}

type ids interface {
	Next(ctx context.Context) (int64, error)
}

type mailQueue interface {
	Enqueue(msg mail.Message)
}

// Service is the accounts use-case layer.
type Service struct {
	users     accountRepo
	sessions  sessions
	hasher    *Hasher
	ids       ids
	limiter   limiter
	mail      mailQueue
	webOrigin string
	logger    *slog.Logger
	now       func() time.Time
}

// NewService wires dependencies. logger and mail may be nil.
func NewService(users accountRepo, sessions sessions, hasher *Hasher, ids ids, limiter limiter, mailer mailQueue, webOrigin string, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		users:     users,
		sessions:  sessions,
		hasher:    hasher,
		ids:       ids,
		limiter:   limiter,
		mail:      mailer,
		webOrigin: webOrigin,
		logger:    logger,
		now:       time.Now,
	}
}

// Register creates a user and session.
func (s *Service) Register(ctx context.Context, in RegisterInput) (SessionUser, error) {
	if err := s.hitLimit(ctx, "register", in.IP, in.Email, registerIPLimit, registerEmailLim); err != nil {
		return SessionUser{}, err
	}
	email, err := normalizeEmail(in.Email)
	if err != nil {
		return SessionUser{}, err
	}
	if err := validatePassword(in.Password, email); err != nil {
		return SessionUser{}, err
	}
	name, err := normalizeDisplayName(in.DisplayName)
	if err != nil {
		return SessionUser{}, err
	}
	if !in.TermsAccepted {
		return SessionUser{}, ErrTermsRequired
	}
	now := s.now().UTC()
	if err := checkAge(in.DateOfBirth, now); err != nil {
		return SessionUser{}, err
	}

	hash, err := s.hasher.Hash(ctx, in.Password)
	if err != nil {
		return SessionUser{}, err
	}
	uid, err := s.ids.Next(ctx)
	if err != nil {
		return SessionUser{}, fmt.Errorf("auth register id: %w", err)
	}
	u := User{
		ID:              uid,
		Email:           email,
		DisplayName:     name,
		PasswordHash:    hash,
		DateOfBirth:     in.DateOfBirth.UTC().Truncate(24 * time.Hour),
		TermsAcceptedAt: now,
		CreatedAt:       now,
	}
	if err := s.users.Insert(ctx, u); err != nil {
		return SessionUser{}, err
	}
	sid, err := s.startSession(ctx, u.ID, in.UserAgent, in.IP)
	if err != nil {
		return SessionUser{}, err
	}
	if err := s.issueAndEnqueueVerify(ctx, u); err != nil {
		s.logger.Error("auth verify email enqueue", slog.String("error", err.Error()), slog.Int64("user_id", u.ID))
	}
	u.PasswordHash = ""
	return SessionUser{User: u, SessionID: sid}, nil
}

// Login authenticates email+password. Unknown email and bad password are the
// same ErrInvalidCredentials.
func (s *Service) Login(ctx context.Context, in LoginInput) (SessionUser, error) {
	if err := s.hitLimit(ctx, "login", in.IP, in.Email, loginLimit, loginLimit); err != nil {
		return SessionUser{}, err
	}
	email, err := normalizeEmail(in.Email)
	if err != nil {
		// Do not enumerate via validation of well-formedness after rate limit:
		// still 401 for "looks like a login" with any password.
		if in.Password == "" {
			return SessionUser{}, ValidationError{Msg: "password is required"}
		}
		_ = s.hasher.CompareDummy(ctx, in.Password)
		return SessionUser{}, ErrInvalidCredentials
	}
	if in.Password == "" {
		return SessionUser{}, ValidationError{Msg: "password is required"}
	}
	u, err := s.users.ByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			_ = s.hasher.CompareDummy(ctx, in.Password)
			return SessionUser{}, ErrInvalidCredentials
		}
		return SessionUser{}, err
	}
	ok, err := s.hasher.Compare(ctx, u.PasswordHash, in.Password)
	if err != nil {
		return SessionUser{}, err
	}
	if !ok {
		return SessionUser{}, ErrInvalidCredentials
	}
	sid, err := s.startSession(ctx, u.ID, in.UserAgent, in.IP)
	if err != nil {
		return SessionUser{}, err
	}
	u.PasswordHash = ""
	return SessionUser{User: u, SessionID: sid}, nil
}

// Logout revokes the session. Missing sessions succeed (idempotent).
func (s *Service) Logout(ctx context.Context, sessionID string) error {
	return s.sessions.Delete(ctx, sessionID)
}

// Me loads the current user from a session id.
func (s *Service) Me(ctx context.Context, sessionID string) (User, error) {
	rec, err := s.sessions.GetAndTouch(ctx, sessionID)
	if err != nil {
		if errors.Is(err, session.ErrNotFound) {
			return User{}, ErrUnauthenticated
		}
		return User{}, fmt.Errorf("auth me session: %w", err)
	}
	u, err := s.users.ByID(ctx, rec.UserID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return User{}, ErrUnauthenticated
		}
		return User{}, err
	}
	u.PasswordHash = ""
	return u, nil
}

// VerifyEmail consumes a one-time token and marks the user verified.
func (s *Service) VerifyEmail(ctx context.Context, token, ip string) error {
	if err := s.hitLimit(ctx, "verify", ip, "", verifyIPLimit, 0); err != nil {
		return err
	}
	hash, err := HashToken(token)
	if err != nil {
		return err
	}
	if _, err := s.users.ConsumeVerifyToken(ctx, hash, s.now().UTC()); err != nil {
		return err
	}
	return nil
}

// ResendVerification issues a new token and enqueues mail. Already-verified
// users get a no-op success. Token persistence errors fail the request; SMTP
// errors are logged inside the mail queue and still succeed.
func (s *Service) ResendVerification(ctx context.Context, sessionID, ip string) error {
	u, err := s.Me(ctx, sessionID)
	if err != nil {
		return err
	}
	if err := s.hitLimitUser(ctx, "resend", ip, u.ID, resendLimit); err != nil {
		return err
	}
	if u.EmailVerified {
		return nil
	}
	return s.issueAndEnqueueVerify(ctx, u)
}

func (s *Service) issueAndEnqueueVerify(ctx context.Context, u User) error {
	raw, hash, err := NewVerifyToken()
	if err != nil {
		return err
	}
	tid, err := s.ids.Next(ctx)
	if err != nil {
		return fmt.Errorf("auth verify token id: %w", err)
	}
	now := s.now().UTC()
	if err := s.users.IssueToken(ctx, AuthToken{
		ID:        tid,
		UserID:    u.ID,
		Purpose:   PurposeVerifyEmail,
		Hash:      hash,
		ExpiresAt: now.Add(verifyTTL),
		CreatedAt: now,
	}); err != nil {
		return err
	}
	s.enqueueVerifyMail(u, raw)
	return nil
}

func (s *Service) enqueueVerifyMail(u User, raw string) {
	if s.mail == nil {
		return
	}
	name := sanitizeLine(u.DisplayName)
	link := strings.TrimRight(s.webOrigin, "/") + "/verify-email?token=" + raw
	s.mail.Enqueue(mail.Message{
		UserID:  u.ID,
		To:      u.Email,
		Subject: "Verify your Friendfold email",
		Body: fmt.Sprintf("Hi %s,\n\nVerify your Friendfold email by opening this link:\n\n%s\n\nThis link expires in 24 hours. If you did not create an account, ignore this email.\n",
			name, link),
	})
}

func sanitizeLine(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	return strings.ReplaceAll(s, "\n", " ")
}

func (s *Service) startSession(ctx context.Context, userID int64, userAgent, ip string) (string, error) {
	sid, err := s.sessions.Create(ctx, session.Record{
		UserID:    userID,
		UserAgent: userAgent,
		IP:        ip,
	})
	if err != nil {
		return "", fmt.Errorf("auth session: %w", err)
	}
	return sid, nil
}

func (s *Service) hitLimit(ctx context.Context, action, ip, email string, ipLimit, emailLimit int) error {
	if s.limiter == nil {
		return nil
	}
	if ip != "" {
		res, err := s.limiter.Allow(ctx, "rate:auth:"+action+":ip:"+ip, ipLimit, authWindow)
		if err != nil {
			s.logger.Error("auth rate limit", slog.String("error", err.Error()), slog.String("action", action))
		} else if !res.Allowed {
			return RateLimitError{RetryAfter: res.RetryAfter}
		}
	}
	if strings.TrimSpace(email) != "" {
		res, err := s.limiter.Allow(ctx, "rate:auth:"+action+":email:"+emailHash(email), emailLimit, authWindow)
		if err != nil {
			s.logger.Error("auth rate limit", slog.String("error", err.Error()), slog.String("action", action))
		} else if !res.Allowed {
			return RateLimitError{RetryAfter: res.RetryAfter}
		}
	}
	return nil
}

func (s *Service) hitLimitUser(ctx context.Context, action, ip string, userID int64, limit int) error {
	if err := s.hitLimit(ctx, action, ip, "", limit, 0); err != nil {
		return err
	}
	if s.limiter == nil || userID == 0 {
		return nil
	}
	res, err := s.limiter.Allow(ctx, "rate:auth:"+action+":user:"+strconv.FormatInt(userID, 10), limit, authWindow)
	if err != nil {
		s.logger.Error("auth rate limit", slog.String("error", err.Error()), slog.String("action", action))
		return nil
	}
	if !res.Allowed {
		return RateLimitError{RetryAfter: res.RetryAfter}
	}
	return nil
}

func emailHash(email string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	return hex.EncodeToString(sum[:])
}

func normalizeEmail(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if len(s) < 3 || len(s) > 254 {
		return "", ValidationError{Msg: "email is invalid"}
	}
	at := strings.LastIndex(s, "@")
	if at < 1 || at == len(s)-1 || strings.ContainsAny(s, " \t\r\n") {
		return "", ValidationError{Msg: "email is invalid"}
	}
	return s, nil
}

func normalizeDisplayName(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	n := utf8.RuneCountInString(s)
	if n < 1 || n > maxDisplayRunes {
		return "", ValidationError{Msg: "display name must be 1–80 characters"}
	}
	return s, nil
}

func validatePassword(password, email string) error {
	n := len(password)
	if n < minPasswordBytes || n > maxPasswordBytes {
		return ValidationError{Msg: "password must be 8–72 bytes"}
	}
	if strings.EqualFold(password, email) {
		return ValidationError{Msg: "password must not be the email address"}
	}
	return nil
}

func checkAge(dob, now time.Time) error {
	if dob.IsZero() {
		return ValidationError{Msg: "date of birth is required"}
	}
	dob = dob.UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if dob.After(today) {
		return ValidationError{Msg: "date of birth is invalid"}
	}
	if ageYears(dob, today) < minAgeYears {
		return ErrUnderage
	}
	return nil
}

func ageYears(born, today time.Time) int {
	age := today.Year() - born.Year()
	anniv := time.Date(today.Year(), born.Month(), born.Day(), 0, 0, 0, 0, time.UTC)
	if today.Before(anniv) {
		age--
	}
	return age
}

// Ensure id.Generator satisfies ids at compile time.
var _ ids = (*id.Generator)(nil)

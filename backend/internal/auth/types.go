package auth

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrInvalidCredentials = errors.New("auth: invalid credentials")
	ErrEmailTaken         = errors.New("auth: email taken")
	ErrUnderage           = errors.New("auth: underage")
	ErrTermsRequired      = errors.New("auth: terms required")
	ErrValidation         = errors.New("auth: validation")
	ErrUnauthenticated    = errors.New("auth: unauthenticated")
	ErrRateLimited        = errors.New("auth: rate limited")
	ErrNotFound           = errors.New("auth: not found")
)

// ValidationError is a 400 validation_error with a public message.
type ValidationError struct {
	Msg string
}

func (e ValidationError) Error() string { return e.Msg }
func (e ValidationError) Unwrap() error { return ErrValidation }

// RateLimitError is a 429 with Retry-After.
type RateLimitError struct {
	RetryAfter time.Duration
}

func (e RateLimitError) Error() string {
	return fmt.Sprintf("auth: rate limited, retry after %s", e.RetryAfter)
}
func (e RateLimitError) Unwrap() error { return ErrRateLimited }

// User is the account aggregate. Date of birth is never serialized on /me.
type User struct {
	ID              int64
	Email           string
	EmailVerified   bool
	DisplayName     string
	PasswordHash    string
	DateOfBirth     time.Time
	TermsAcceptedAt time.Time
	CreatedAt       time.Time
}

// RegisterInput is the register use case.
type RegisterInput struct {
	Email         string
	Password      string
	DisplayName   string
	DateOfBirth   time.Time
	TermsAccepted bool
	UserAgent     string
	IP            string
}

// LoginInput is the login use case.
type LoginInput struct {
	Email     string
	Password  string
	UserAgent string
	IP        string
}

// SessionUser is the authenticated principal plus the session id (never logged).
type SessionUser struct {
	User      User
	SessionID string
}

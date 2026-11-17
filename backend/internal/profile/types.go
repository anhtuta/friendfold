package profile

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrValidation       = errors.New("profile: validation")
	ErrNotFound         = errors.New("profile: not found")
	ErrUnauthenticated  = errors.New("profile: unauthenticated")
	ErrUsernameTaken    = errors.New("profile: username taken")
	ErrUsernameReserved = errors.New("profile: username reserved")
	ErrUsernameInvalid  = errors.New("profile: username invalid")
	ErrUsernameCooldown = errors.New("profile: username cooldown")
	ErrRateLimited      = errors.New("profile: rate limited")
	ErrForbidden        = errors.New("profile: forbidden")
)

// ValidationError is a 400 validation_error.
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
	return fmt.Sprintf("profile: rate limited, retry after %s", e.RetryAfter)
}
func (e RateLimitError) Unwrap() error { return ErrRateLimited }

// Account is the identity slice profile needs from auth (no DOB, no hash).
type Account struct {
	ID            int64
	Email         string
	EmailVerified bool
	DisplayName   string
}

// Record is a profiles row.
type Record struct {
	UserID            int64
	Username          string // empty if unset
	UsernameChangedAt time.Time
	Bio               string
	AvatarObjectKey   string
	CoverObjectKey    string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Public is the JSON public profile (no email).
type Public struct {
	ID          int64
	Username    string
	DisplayName string
	Bio         string
	AvatarURL   string
	CoverURL    string
}

// Me is the owner payload.
type Me struct {
	Public
	Email              string
	EmailVerified      bool
	UsernameChangedAt  time.Time
	HasUsernameChanged bool
}

// PatchInput is PATCH /v1/me/profile.
type PatchInput struct {
	UserID      int64
	IP          string
	DisplayName *string
	Bio         *string
	Username    *string
}

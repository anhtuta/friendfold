package profile

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/anhtuta/friendfold/backend/internal/platform/blob"
	"github.com/anhtuta/friendfold/backend/internal/platform/ratelimit"
)

const (
	patchLimit  = 30
	publicLimit = 120
	limitWindow = 15 * time.Minute
)

type accounts interface {
	ByID(ctx context.Context, id int64) (Account, error)
	UpdateDisplayName(ctx context.Context, id int64, name string, now time.Time) error
}

type blocks interface {
	Blocked(ctx context.Context, viewerID, ownerID int64) (bool, error)
}

type limiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (ratelimit.Result, error)
}

// NoopBlocks is the feature-04 placeholder: nobody is blocked.
type NoopBlocks struct{}

// Blocked always returns false.
func (NoopBlocks) Blocked(context.Context, int64, int64) (bool, error) { return false, nil }

// Service is the profile use-case layer.
type Service struct {
	repo       repo
	accounts   accounts
	blocks     blocks
	cache      *publicCache
	limiter    limiter
	logger     *slog.Logger
	publicBase string
	now        func() time.Time
}

// NewService wires dependencies. rdb, limiter, blocks, and logger may be nil.
func NewService(repo repo, accounts accounts, blocks blocks, rdb redis.Cmdable, limiter limiter, publicBase string, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	if blocks == nil {
		blocks = NoopBlocks{}
	}
	var cache *publicCache
	if rdb != nil {
		cache = &publicCache{rdb: rdb}
	}
	return &Service{
		repo:       repo,
		accounts:   accounts,
		blocks:     blocks,
		cache:      cache,
		limiter:    limiter,
		logger:     logger,
		publicBase: publicBase,
		now:        time.Now,
	}
}

// GetMe loads or creates the owner profile.
func (s *Service) GetMe(ctx context.Context, userID int64) (Me, error) {
	acct, err := s.accounts.ByID(ctx, userID)
	if err != nil {
		return Me{}, err
	}
	rec, err := s.repo.Upsert(ctx, userID, s.now().UTC())
	if err != nil {
		return Me{}, err
	}
	return s.toMe(acct, rec), nil
}

// PatchMe updates owner text fields.
func (s *Service) PatchMe(ctx context.Context, in PatchInput) (Me, error) {
	if in.DisplayName == nil && in.Bio == nil && in.Username == nil {
		return Me{}, ValidationError{Msg: "at least one field is required"}
	}
	if err := s.hitUserLimit(ctx, "patch", in.UserID, in.IP, patchLimit); err != nil {
		return Me{}, err
	}
	acct, err := s.accounts.ByID(ctx, in.UserID)
	if err != nil {
		return Me{}, err
	}
	now := s.now().UTC()
	rec, err := s.repo.Upsert(ctx, in.UserID, now)
	if err != nil {
		return Me{}, err
	}
	oldUsername := rec.Username

	if in.DisplayName != nil {
		name, err := normalizeDisplayName(*in.DisplayName)
		if err != nil {
			return Me{}, err
		}
		if err := s.accounts.UpdateDisplayName(ctx, in.UserID, name, now); err != nil {
			return Me{}, err
		}
		acct.DisplayName = name
	}
	if in.Bio != nil {
		bio, err := normalizeBio(*in.Bio)
		if err != nil {
			return Me{}, err
		}
		rec, err = s.repo.UpdateBio(ctx, in.UserID, bio, now)
		if err != nil {
			return Me{}, err
		}
	}
	if in.Username != nil {
		name, err := normalizeUsername(*in.Username)
		if err != nil {
			return Me{}, err
		}
		if rec.Username != name {
			if !rec.UsernameChangedAt.IsZero() && now.Sub(rec.UsernameChangedAt) < usernameCooldown {
				return Me{}, ErrUsernameCooldown
			}
			rec, err = s.repo.UpdateUsername(ctx, in.UserID, name, now)
			if err != nil {
				return Me{}, err
			}
		}
	}
	s.cache.invalidate(ctx, in.UserID, oldUsername, rec.Username)
	return s.toMe(acct, rec), nil
}

// GetPublic returns a public profile. viewerID 0 is anonymous.
func (s *Service) GetPublic(ctx context.Context, username string, viewerID int64, ip string) (Public, error) {
	if err := s.hitIPLimit(ctx, "get", ip, publicLimit); err != nil {
		return Public{}, err
	}
	name, err := normalizeUsername(username)
	if err != nil {
		if errors.Is(err, ErrUsernameReserved) {
			// Reserved names that were never claimed are indistinguishable from
			// missing users.
			return Public{}, ErrNotFound
		}
		return Public{}, err
	}
	if cached, ok, err := s.cache.get(ctx, name); err != nil {
		s.logger.Error("profile cache get", slog.String("error", err.Error()))
	} else if ok {
		if err := s.denyIfBlocked(ctx, viewerID, cached.ID); err != nil {
			return Public{}, err
		}
		if _, err := s.accounts.ByID(ctx, cached.ID); err != nil {
			if errors.Is(err, ErrNotFound) {
				s.cache.invalidate(ctx, cached.ID, name)
				return Public{}, ErrNotFound
			}
			return Public{}, err
		}
		return cached, nil
	}
	rec, err := s.repo.ByUsername(ctx, name)
	if err != nil {
		return Public{}, err
	}
	acct, err := s.accounts.ByID(ctx, rec.UserID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Public{}, ErrNotFound
		}
		return Public{}, err
	}
	if err := s.denyIfBlocked(ctx, viewerID, rec.UserID); err != nil {
		return Public{}, err
	}
	pub := s.toPublic(acct, rec)
	if err := s.cache.set(ctx, pub); err != nil {
		s.logger.Error("profile cache set", slog.String("error", err.Error()))
	}
	return pub, nil
}

func (s *Service) denyIfBlocked(ctx context.Context, viewerID, ownerID int64) error {
	if viewerID == 0 || viewerID == ownerID {
		return nil
	}
	blocked, err := s.blocks.Blocked(ctx, viewerID, ownerID)
	if err != nil {
		return fmt.Errorf("profile block check: %w", err)
	}
	if blocked {
		return ErrNotFound
	}
	blocked, err = s.blocks.Blocked(ctx, ownerID, viewerID)
	if err != nil {
		return fmt.Errorf("profile block check: %w", err)
	}
	if blocked {
		return ErrNotFound
	}
	return nil
}

func (s *Service) toPublic(acct Account, rec Record) Public {
	return Public{
		ID:          acct.ID,
		Username:    rec.Username,
		DisplayName: acct.DisplayName,
		Bio:         rec.Bio,
		AvatarURL:   blob.PublicURL(s.publicBase, rec.AvatarObjectKey),
		CoverURL:    blob.PublicURL(s.publicBase, rec.CoverObjectKey),
	}
}

func (s *Service) toMe(acct Account, rec Record) Me {
	return Me{
		Public:             s.toPublic(acct, rec),
		Email:              acct.Email,
		EmailVerified:      acct.EmailVerified,
		UsernameChangedAt:  rec.UsernameChangedAt,
		HasUsernameChanged: !rec.UsernameChangedAt.IsZero(),
	}
}

func (s *Service) hitUserLimit(ctx context.Context, action string, userID int64, ip string, limit int) error {
	_ = ip
	return s.allow(ctx, "rate:profile:"+action+":"+strconv.FormatInt(userID, 10), limit)
}

func (s *Service) hitIPLimit(ctx context.Context, action, ip string, limit int) error {
	if ip == "" {
		ip = "unknown"
	}
	return s.allow(ctx, "rate:profile:"+action+":"+ip, limit)
}

func (s *Service) allow(ctx context.Context, key string, limit int) error {
	if s.limiter == nil {
		return nil
	}
	res, err := s.limiter.Allow(ctx, key, limit, limitWindow)
	if err != nil {
		s.logger.Error("profile rate limit", slog.String("error", err.Error()), slog.String("key", key))
		return nil
	}
	if !res.Allowed {
		return RateLimitError{RetryAfter: res.RetryAfter}
	}
	return nil
}

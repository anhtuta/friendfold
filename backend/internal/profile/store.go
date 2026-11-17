package profile

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/anhtuta/friendfold/backend/internal/profile/sqlc"
)

type repo interface {
	Upsert(ctx context.Context, userID int64, now time.Time) (Record, error)
	ByUserID(ctx context.Context, userID int64) (Record, error)
	ByUsername(ctx context.Context, username string) (Record, error)
	UpdateBio(ctx context.Context, userID int64, bio string, now time.Time) (Record, error)
	UpdateUsername(ctx context.Context, userID int64, username string, now time.Time) (Record, error)
	SetAvatar(ctx context.Context, userID int64, key string, now time.Time) (Record, error)
	SetCover(ctx context.Context, userID int64, key string, now time.Time) (Record, error)
	InsertMedia(ctx context.Context, m Media) error
}

// Store is the Postgres implementation of repo.
type Store struct {
	q *sqlc.Queries
}

// NewStore wraps a pgx pool.
func NewStore(db sqlc.DBTX) *Store {
	return &Store{q: sqlc.New(db)}
}

func (s *Store) Upsert(ctx context.Context, userID int64, now time.Time) (Record, error) {
	row, err := s.q.UpsertProfile(ctx, sqlc.UpsertProfileParams{
		UserID:    userID,
		CreatedAt: pgTime(now),
	})
	if err != nil {
		return Record{}, fmt.Errorf("profile upsert: %w", err)
	}
	return recordFromRow(row), nil
}

func (s *Store) ByUserID(ctx context.Context, userID int64) (Record, error) {
	row, err := s.q.GetProfileByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Record{}, ErrNotFound
		}
		return Record{}, fmt.Errorf("profile by user: %w", err)
	}
	return recordFromRow(row), nil
}

func (s *Store) ByUsername(ctx context.Context, username string) (Record, error) {
	row, err := s.q.GetProfileByUsername(ctx, pgtype.Text{String: username, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Record{}, ErrNotFound
		}
		return Record{}, fmt.Errorf("profile by username: %w", err)
	}
	return recordFromRow(row), nil
}

func (s *Store) UpdateBio(ctx context.Context, userID int64, bio string, now time.Time) (Record, error) {
	row, err := s.q.UpdateBio(ctx, sqlc.UpdateBioParams{
		UserID:    userID,
		Bio:       bio,
		UpdatedAt: pgTime(now),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Record{}, ErrNotFound
		}
		return Record{}, fmt.Errorf("profile update bio: %w", err)
	}
	return recordFromRow(row), nil
}

func (s *Store) UpdateUsername(ctx context.Context, userID int64, username string, now time.Time) (Record, error) {
	row, err := s.q.UpdateUsername(ctx, sqlc.UpdateUsernameParams{
		UserID:            userID,
		Username:          pgtype.Text{String: username, Valid: true},
		UsernameChangedAt: pgTime(now),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Record{}, ErrUsernameTaken
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return Record{}, ErrNotFound
		}
		return Record{}, fmt.Errorf("profile update username: %w", err)
	}
	return recordFromRow(row), nil
}

func (s *Store) SetAvatar(ctx context.Context, userID int64, key string, now time.Time) (Record, error) {
	row, err := s.q.SetAvatarKey(ctx, sqlc.SetAvatarKeyParams{
		UserID:          userID,
		AvatarObjectKey: pgText(key),
		UpdatedAt:       pgTime(now),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Record{}, ErrNotFound
		}
		return Record{}, fmt.Errorf("profile set avatar: %w", err)
	}
	return recordFromRow(row), nil
}

func (s *Store) SetCover(ctx context.Context, userID int64, key string, now time.Time) (Record, error) {
	row, err := s.q.SetCoverKey(ctx, sqlc.SetCoverKeyParams{
		UserID:         userID,
		CoverObjectKey: pgText(key),
		UpdatedAt:      pgTime(now),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Record{}, ErrNotFound
		}
		return Record{}, fmt.Errorf("profile set cover: %w", err)
	}
	return recordFromRow(row), nil
}

func (s *Store) InsertMedia(ctx context.Context, m Media) error {
	_, err := s.q.InsertMediaObject(ctx, sqlc.InsertMediaObjectParams{
		ID:        m.ID,
		UserID:    m.UserID,
		Kind:      m.Kind,
		ObjectKey: m.ObjectKey,
		Mime:      m.MIME,
		Bytes:     int32(m.Bytes),
		Width:     int32(m.Width),
		Height:    int32(m.Height),
		CreatedAt: pgTime(m.CreatedAt),
	})
	if err != nil {
		return fmt.Errorf("profile insert media: %w", err)
	}
	return nil
}

func recordFromRow(row sqlc.Profile) Record {
	return Record{
		UserID:            row.UserID,
		Username:          row.Username.String,
		UsernameChangedAt: row.UsernameChangedAt.Time,
		Bio:               row.Bio,
		AvatarObjectKey:   row.AvatarObjectKey.String,
		CoverObjectKey:    row.CoverObjectKey.String,
		CreatedAt:         row.CreatedAt.Time,
		UpdatedAt:         row.UpdatedAt.Time,
	}
}

func pgTime(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t.UTC(), Valid: true}
}

func pgText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

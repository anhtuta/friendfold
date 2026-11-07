package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/anhtuta/friendfold/backend/internal/auth/sqlc"
)

type accountRepo interface {
	Insert(ctx context.Context, u User) error
	ByEmail(ctx context.Context, email string) (User, error)
	ByID(ctx context.Context, id int64) (User, error)
	IssueToken(ctx context.Context, tok AuthToken) error
	ConsumeVerifyToken(ctx context.Context, hash []byte, now time.Time) (userID int64, err error)
	LiveResetUser(ctx context.Context, hash []byte, now time.Time) (User, error)
	ConsumeResetAndSetPassword(ctx context.Context, hash []byte, passwordHash string, now time.Time) (userID int64, err error)
	UpdatePasswordHash(ctx context.Context, userID int64, passwordHash string, now time.Time) error
}

// Store is the Postgres implementation of accountRepo (sqlc).
type Store struct {
	db sqlc.DBTX
	q  *sqlc.Queries
}

// NewStore wraps a pgx pool or tx.
func NewStore(db sqlc.DBTX) *Store {
	return &Store{db: db, q: sqlc.New(db)}
}

// Insert writes a new user. Duplicate live email becomes ErrEmailTaken.
func (s *Store) Insert(ctx context.Context, u User) error {
	_, err := s.q.InsertUser(ctx, sqlc.InsertUserParams{
		ID:              u.ID,
		Email:           u.Email,
		PasswordHash:    pgtype.Text{String: u.PasswordHash, Valid: u.PasswordHash != ""},
		DisplayName:     u.DisplayName,
		DateOfBirth:     pgDate(u.DateOfBirth),
		TermsAcceptedAt: pgTime(u.TermsAcceptedAt),
		CreatedAt:       pgTime(u.CreatedAt),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrEmailTaken
		}
		return fmt.Errorf("auth insert user: %w", err)
	}
	return nil
}

// ByEmail loads a non-deleted user by lowercased email.
func (s *Store) ByEmail(ctx context.Context, email string) (User, error) {
	row, err := s.q.GetUserByEmail(ctx, email)
	if err != nil {
		return User{}, mapNotFound(err, "auth get user by email")
	}
	return userFromRow(row), nil
}

// ByID loads a non-deleted user.
func (s *Store) ByID(ctx context.Context, id int64) (User, error) {
	row, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		return User{}, mapNotFound(err, "auth get user by id")
	}
	return userFromRow(row), nil
}

func mapNotFound(err error, op string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return fmt.Errorf("%s: %w", op, err)
}

// IssueToken invalidates unused tokens of the same purpose, then inserts tok.
func (s *Store) IssueToken(ctx context.Context, tok AuthToken) error {
	return s.inTx(ctx, func(q *sqlc.Queries) error {
		if err := q.InvalidateUnusedAuthTokens(ctx, sqlc.InvalidateUnusedAuthTokensParams{
			UserID:     tok.UserID,
			Purpose:    tok.Purpose,
			ConsumedAt: pgTime(tok.CreatedAt),
		}); err != nil {
			return fmt.Errorf("auth invalidate tokens: %w", err)
		}
		_, err := q.InsertAuthToken(ctx, sqlc.InsertAuthTokenParams{
			ID:        tok.ID,
			UserID:    tok.UserID,
			Purpose:   tok.Purpose,
			TokenHash: tok.Hash,
			ExpiresAt: pgTime(tok.ExpiresAt),
			CreatedAt: pgTime(tok.CreatedAt),
		})
		if err != nil {
			return fmt.Errorf("auth insert token: %w", err)
		}
		return nil
	})
}

// ConsumeVerifyToken marks a verify_email token used and sets email_verified_at.
func (s *Store) ConsumeVerifyToken(ctx context.Context, hash []byte, now time.Time) (int64, error) {
	var userID int64
	err := s.inTx(ctx, func(q *sqlc.Queries) error {
		row, err := q.ConsumeAuthToken(ctx, sqlc.ConsumeAuthTokenParams{
			TokenHash:  hash,
			ConsumedAt: pgTime(now),
			Purpose:    PurposeVerifyEmail,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrInvalidToken
			}
			return fmt.Errorf("auth consume token: %w", err)
		}
		if _, err := q.MarkEmailVerified(ctx, sqlc.MarkEmailVerifiedParams{
			ID:              row.UserID,
			EmailVerifiedAt: pgTime(now),
		}); err != nil {
			return fmt.Errorf("auth mark verified: %w", err)
		}
		userID = row.UserID
		return nil
	})
	if err != nil {
		return 0, err
	}
	return userID, nil
}

// LiveResetUser returns the user for an unused, unexpired reset_password token
// without consuming it.
func (s *Store) LiveResetUser(ctx context.Context, hash []byte, now time.Time) (User, error) {
	row, err := s.q.GetLiveAuthToken(ctx, sqlc.GetLiveAuthTokenParams{
		TokenHash: hash,
		Purpose:   PurposeResetPassword,
		ExpiresAt: pgTime(now),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrInvalidResetToken
		}
		return User{}, fmt.Errorf("auth live reset token: %w", err)
	}
	u, err := s.ByID(ctx, row.UserID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return User{}, ErrInvalidResetToken
		}
		return User{}, err
	}
	return u, nil
}

// ConsumeResetAndSetPassword consumes a reset_password token and writes the
// new hash in one transaction.
func (s *Store) ConsumeResetAndSetPassword(ctx context.Context, hash []byte, passwordHash string, now time.Time) (int64, error) {
	var userID int64
	err := s.inTx(ctx, func(q *sqlc.Queries) error {
		row, err := q.ConsumeAuthToken(ctx, sqlc.ConsumeAuthTokenParams{
			TokenHash:  hash,
			ConsumedAt: pgTime(now),
			Purpose:    PurposeResetPassword,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrInvalidResetToken
			}
			return fmt.Errorf("auth consume reset token: %w", err)
		}
		if err := q.UpdatePasswordHash(ctx, sqlc.UpdatePasswordHashParams{
			ID:           row.UserID,
			PasswordHash: pgtype.Text{String: passwordHash, Valid: passwordHash != ""},
			UpdatedAt:    pgTime(now),
		}); err != nil {
			return fmt.Errorf("auth update password: %w", err)
		}
		userID = row.UserID
		return nil
	})
	if err != nil {
		return 0, err
	}
	return userID, nil
}

// UpdatePasswordHash writes a new argon2id hash for a live user.
func (s *Store) UpdatePasswordHash(ctx context.Context, userID int64, passwordHash string, now time.Time) error {
	if err := s.q.UpdatePasswordHash(ctx, sqlc.UpdatePasswordHashParams{
		ID:           userID,
		PasswordHash: pgtype.Text{String: passwordHash, Valid: passwordHash != ""},
		UpdatedAt:    pgTime(now),
	}); err != nil {
		return fmt.Errorf("auth update password: %w", err)
	}
	return nil
}

func (s *Store) inTx(ctx context.Context, fn func(*sqlc.Queries) error) error {
	beginner, ok := s.db.(interface {
		Begin(context.Context) (pgx.Tx, error)
	})
	if !ok {
		return fn(s.q)
	}
	tx, err := beginner.Begin(ctx)
	if err != nil {
		return fmt.Errorf("auth begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(sqlc.New(tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("auth commit: %w", err)
	}
	return nil
}

func userFromRow(row sqlc.User) User {
	return User{
		ID:              row.ID,
		Email:           row.Email,
		EmailVerified:   row.EmailVerifiedAt.Valid,
		DisplayName:     row.DisplayName,
		PasswordHash:    row.PasswordHash.String,
		DateOfBirth:     row.DateOfBirth.Time,
		TermsAcceptedAt: row.TermsAcceptedAt.Time,
		CreatedAt:       row.CreatedAt.Time,
	}
}

func pgDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t.UTC(), Valid: true}
}

func pgTime(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t.UTC(), Valid: true}
}

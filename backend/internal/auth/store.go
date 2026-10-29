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

type userRepo interface {
	Insert(ctx context.Context, u User) error
	ByEmail(ctx context.Context, email string) (User, error)
	ByID(ctx context.Context, id int64) (User, error)
}

// Store is the Postgres implementation of userRepo (sqlc).
type Store struct {
	q *sqlc.Queries
}

// NewStore wraps a pgx pool or tx.
func NewStore(db sqlc.DBTX) *Store {
	return &Store{q: sqlc.New(db)}
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

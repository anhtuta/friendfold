package main

import (
	"context"
	"errors"
	"time"

	"github.com/anhtuta/friendfold/backend/internal/auth"
	"github.com/anhtuta/friendfold/backend/internal/profile"
)

// profileAccounts adapts auth.Store to profile's Account interface so profile
// does not SELECT from users.
type profileAccounts struct {
	users *auth.Store
}

func (a profileAccounts) ByID(ctx context.Context, id int64) (profile.Account, error) {
	u, err := a.users.ByID(ctx, id)
	if err != nil {
		if errors.Is(err, auth.ErrNotFound) {
			return profile.Account{}, profile.ErrNotFound
		}
		return profile.Account{}, err
	}
	return profile.Account{
		ID:            u.ID,
		Email:         u.Email,
		EmailVerified: u.EmailVerified,
		DisplayName:   u.DisplayName,
	}, nil
}

func (a profileAccounts) UpdateDisplayName(ctx context.Context, id int64, name string, now time.Time) error {
	err := a.users.UpdateDisplayName(ctx, id, name, now)
	if errors.Is(err, auth.ErrNotFound) {
		return profile.ErrNotFound
	}
	return err
}

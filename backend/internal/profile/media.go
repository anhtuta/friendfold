package profile

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"

	"github.com/anhtuta/friendfold/backend/internal/platform/blob"
)

// UploadImage transcodes file bytes and stores them as the owner's avatar or cover.
func (s *Service) UploadImage(ctx context.Context, userID int64, ip, kind string, raw []byte) (Me, error) {
	maxLong, maxBytes, limit := kindLimits(kind)
	if maxLong == 0 {
		return Me{}, ValidationError{Msg: "unknown media kind"}
	}
	if len(raw) == 0 {
		return Me{}, ValidationError{Msg: "file is required"}
	}
	if len(raw) > maxBytes {
		return Me{}, ErrTooLarge
	}
	if err := s.hitUserLimit(ctx, kind, userID, ip, limit); err != nil {
		return Me{}, err
	}
	if s.blobs == nil || s.ids == nil {
		return Me{}, fmt.Errorf("profile media: store is not configured")
	}

	acct, err := s.accounts.ByID(ctx, userID)
	if err != nil {
		return Me{}, err
	}
	now := s.now().UTC()
	rec, err := s.repo.Upsert(ctx, userID, now)
	if err != nil {
		return Me{}, err
	}

	tc, err := transcodeImage(raw, maxLong)
	if err != nil {
		return Me{}, err
	}
	oid, err := s.ids.Next(ctx)
	if err != nil {
		return Me{}, fmt.Errorf("profile media id: %w", err)
	}
	key := "profile/" + strconv.FormatInt(userID, 10) + "/" + kind + "/" + strconv.FormatInt(oid, 10) + ".jpg"
	if err := s.blobs.Put(ctx, key, "image/jpeg", bytes.NewReader(tc.JPEG)); err != nil {
		return Me{}, fmt.Errorf("profile media put: %w", err)
	}
	if err := s.repo.InsertMedia(ctx, Media{
		ID:        oid,
		UserID:    userID,
		Kind:      kind,
		ObjectKey: key,
		MIME:      "image/jpeg",
		Bytes:     len(tc.JPEG),
		Width:     tc.Width,
		Height:    tc.Height,
		CreatedAt: now,
	}); err != nil {
		_ = s.blobs.Delete(ctx, key)
		return Me{}, err
	}

	prev := rec.AvatarObjectKey
	if kind == KindCover {
		prev = rec.CoverObjectKey
		rec, err = s.repo.SetCover(ctx, userID, key, now)
	} else {
		rec, err = s.repo.SetAvatar(ctx, userID, key, now)
	}
	if err != nil {
		return Me{}, err
	}
	if prev != "" && prev != key {
		if err := s.blobs.Delete(ctx, prev); err != nil && !errors.Is(err, blob.ErrNotFound) {
			s.logger.Error("profile media delete previous",
				slog.Int64("user_id", userID),
				slog.String("kind", kind),
				slog.String("error", err.Error()),
			)
		}
	}
	s.logger.Info("profile media stored",
		slog.Int64("user_id", userID),
		slog.String("kind", kind),
		slog.Int("bytes", len(tc.JPEG)),
	)
	s.cache.invalidate(ctx, userID, rec.Username)
	return s.toMe(acct, rec), nil
}

// DeleteImage clears avatar or cover and removes the previous object.
func (s *Service) DeleteImage(ctx context.Context, userID int64, ip, kind string) (Me, error) {
	_, _, limit := kindLimits(kind)
	if limit == 0 {
		return Me{}, ValidationError{Msg: "unknown media kind"}
	}
	if err := s.hitUserLimit(ctx, kind, userID, ip, limit); err != nil {
		return Me{}, err
	}
	acct, err := s.accounts.ByID(ctx, userID)
	if err != nil {
		return Me{}, err
	}
	now := s.now().UTC()
	rec, err := s.repo.Upsert(ctx, userID, now)
	if err != nil {
		return Me{}, err
	}
	prev := rec.AvatarObjectKey
	if kind == KindCover {
		prev = rec.CoverObjectKey
		rec, err = s.repo.SetCover(ctx, userID, "", now)
	} else {
		rec, err = s.repo.SetAvatar(ctx, userID, "", now)
	}
	if err != nil {
		return Me{}, err
	}
	if prev != "" && s.blobs != nil {
		if err := s.blobs.Delete(ctx, prev); err != nil && !errors.Is(err, blob.ErrNotFound) {
			s.logger.Error("profile media delete previous",
				slog.Int64("user_id", userID),
				slog.String("kind", kind),
				slog.String("error", err.Error()),
			)
		}
	}
	s.cache.invalidate(ctx, userID, rec.Username)
	return s.toMe(acct, rec), nil
}

// OpenObject streams a stored JPEG. Invalid or missing keys are ErrNotFound.
func (s *Service) OpenObject(ctx context.Context, key string) (io.ReadCloser, string, error) {
	if err := blob.ValidateKey(key); err != nil {
		return nil, "", ErrNotFound
	}
	if s.blobs == nil {
		return nil, "", ErrNotFound
	}
	rc, ct, err := s.blobs.Get(ctx, key)
	if err != nil {
		if errors.Is(err, blob.ErrNotFound) || errors.Is(err, blob.ErrInvalidKey) {
			return nil, "", ErrNotFound
		}
		return nil, "", fmt.Errorf("profile media get: %w", err)
	}
	return rc, ct, nil
}

func kindLimits(kind string) (maxLong, maxBytes, limit int) {
	switch kind {
	case KindAvatar:
		return avatarMaxLong, avatarMaxBytes, avatarLimit
	case KindCover:
		return coverMaxLong, coverMaxBytes, coverLimit
	default:
		return 0, 0, 0
	}
}

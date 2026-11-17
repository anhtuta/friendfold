-- name: UpsertProfile :one
INSERT INTO profiles (
    user_id,
    bio,
    created_at,
    updated_at
) VALUES (
    $1, '', $2, $2
)
ON CONFLICT (user_id) DO UPDATE SET user_id = profiles.user_id
RETURNING user_id, username, username_changed_at, bio, avatar_object_key, cover_object_key, created_at, updated_at;

-- name: GetProfileByUserID :one
SELECT user_id, username, username_changed_at, bio, avatar_object_key, cover_object_key, created_at, updated_at
FROM profiles
WHERE user_id = $1;

-- name: GetProfileByUsername :one
SELECT user_id, username, username_changed_at, bio, avatar_object_key, cover_object_key, created_at, updated_at
FROM profiles
WHERE username = $1;

-- name: UpdateBio :one
UPDATE profiles
SET bio = $2, updated_at = $3
WHERE user_id = $1
RETURNING user_id, username, username_changed_at, bio, avatar_object_key, cover_object_key, created_at, updated_at;

-- name: UpdateUsername :one
UPDATE profiles
SET username = $2, username_changed_at = $3, updated_at = $3
WHERE user_id = $1
RETURNING user_id, username, username_changed_at, bio, avatar_object_key, cover_object_key, created_at, updated_at;

-- name: SetAvatarKey :one
UPDATE profiles
SET avatar_object_key = $2, updated_at = $3
WHERE user_id = $1
RETURNING user_id, username, username_changed_at, bio, avatar_object_key, cover_object_key, created_at, updated_at;

-- name: SetCoverKey :one
UPDATE profiles
SET cover_object_key = $2, updated_at = $3
WHERE user_id = $1
RETURNING user_id, username, username_changed_at, bio, avatar_object_key, cover_object_key, created_at, updated_at;

-- name: InsertMediaObject :one
INSERT INTO media_objects (
    id,
    user_id,
    kind,
    object_key,
    mime,
    bytes,
    width,
    height,
    created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING id, user_id, kind, object_key, mime, bytes, width, height, created_at;

-- name: GetMediaObjectByKey :one
SELECT id, user_id, kind, object_key, mime, bytes, width, height, created_at
FROM media_objects
WHERE object_key = $1;

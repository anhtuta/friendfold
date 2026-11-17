-- +goose Up
-- Feature 03: profiles and media object index. users remains auth-owned.
-- Display name stays on users; profile module talks to it through an interface.

CREATE TABLE profiles (
    user_id BIGINT PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    username TEXT,
    username_changed_at TIMESTAMPTZ,
    bio TEXT NOT NULL DEFAULT '',
    avatar_object_key TEXT,
    cover_object_key TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT profiles_username_len CHECK (
        username IS NULL OR char_length(username) BETWEEN 3 AND 20
    ),
    CONSTRAINT profiles_username_format CHECK (
        username IS NULL OR username ~ '^[a-z0-9_]+$'
    ),
    CONSTRAINT profiles_bio_len CHECK (char_length(bio) <= 640)
);

CREATE UNIQUE INDEX profiles_username_key ON profiles (username) WHERE username IS NOT NULL;

CREATE TABLE media_objects (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    object_key TEXT NOT NULL,
    mime TEXT NOT NULL,
    bytes INT NOT NULL,
    width INT NOT NULL,
    height INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT media_objects_kind_check CHECK (kind IN ('avatar', 'cover')),
    CONSTRAINT media_objects_bytes_pos CHECK (bytes > 0),
    CONSTRAINT media_objects_dims_pos CHECK (width > 0 AND height > 0)
);

CREATE UNIQUE INDEX media_objects_object_key_key ON media_objects (object_key);
CREATE INDEX media_objects_user_created_idx ON media_objects (user_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS media_objects;
DROP TABLE IF EXISTS profiles;

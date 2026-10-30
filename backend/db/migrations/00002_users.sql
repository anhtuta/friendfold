-- +goose Up
-- Feature 02 phase 1: identity root. No oauth_accounts or auth_tokens yet.

CREATE TABLE users (
    id BIGINT PRIMARY KEY,
    email TEXT NOT NULL,
    email_verified_at TIMESTAMPTZ,
    password_hash TEXT,
    display_name TEXT NOT NULL,
    date_of_birth DATE NOT NULL,
    terms_accepted_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT users_display_name_len CHECK (char_length(display_name) BETWEEN 1 AND 80),
    CONSTRAINT users_email_len CHECK (char_length(email) BETWEEN 3 AND 254)
);

-- Unique among non-deleted rows. Email is stored lowercased by the auth module.
CREATE UNIQUE INDEX users_email_live_key ON users (email) WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS users;

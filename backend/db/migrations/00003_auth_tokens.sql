-- +goose Up
-- Feature 02 phase 2: hashed one-time tokens for email verification
-- (reset_password purpose is reserved for phase 3).

CREATE TABLE auth_tokens (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    purpose TEXT NOT NULL,
    token_hash BYTEA NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT auth_tokens_purpose_check CHECK (purpose IN ('verify_email', 'reset_password')),
    CONSTRAINT auth_tokens_hash_len CHECK (octet_length(token_hash) = 32)
);

CREATE UNIQUE INDEX auth_tokens_token_hash_key ON auth_tokens (token_hash);
CREATE UNIQUE INDEX auth_tokens_unused_purpose_key ON auth_tokens (user_id, purpose) WHERE consumed_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS auth_tokens;

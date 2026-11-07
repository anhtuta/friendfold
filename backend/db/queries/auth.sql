-- name: InsertUser :one
INSERT INTO users (
    id,
    email,
    email_verified_at,
    password_hash,
    display_name,
    date_of_birth,
    terms_accepted_at,
    created_at,
    updated_at,
    deleted_at
) VALUES (
    $1, $2, NULL, $3, $4, $5, $6, $7, $7, NULL
)
RETURNING id, email, email_verified_at, password_hash, display_name, date_of_birth, terms_accepted_at, created_at, updated_at, deleted_at;

-- name: GetUserByEmail :one
SELECT id, email, email_verified_at, password_hash, display_name, date_of_birth, terms_accepted_at, created_at, updated_at, deleted_at
FROM users
WHERE email = $1 AND deleted_at IS NULL;

-- name: GetUserByID :one
SELECT id, email, email_verified_at, password_hash, display_name, date_of_birth, terms_accepted_at, created_at, updated_at, deleted_at
FROM users
WHERE id = $1 AND deleted_at IS NULL;

-- name: MarkEmailVerified :execrows
UPDATE users
SET email_verified_at = $2, updated_at = $2
WHERE id = $1 AND deleted_at IS NULL AND email_verified_at IS NULL;

-- name: InsertAuthToken :one
INSERT INTO auth_tokens (
    id,
    user_id,
    purpose,
    token_hash,
    expires_at,
    consumed_at,
    created_at
) VALUES (
    $1, $2, $3, $4, $5, NULL, $6
)
RETURNING id, user_id, purpose, token_hash, expires_at, consumed_at, created_at;

-- name: ConsumeAuthToken :one
UPDATE auth_tokens
SET consumed_at = $2
WHERE token_hash = $1
  AND consumed_at IS NULL
  AND expires_at > $2
RETURNING id, user_id, purpose, token_hash, expires_at, consumed_at, created_at;

-- name: InvalidateUnusedAuthTokens :exec
UPDATE auth_tokens
SET consumed_at = $3
WHERE user_id = $1
  AND purpose = $2
  AND consumed_at IS NULL;

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

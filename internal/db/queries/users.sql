-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at;

-- name: CreateUser :one
INSERT INTO users (username, email, password_hash)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateUserPublicKey :one
UPDATE users SET public_key = $1, updated_at = NOW()
WHERE id = $2
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

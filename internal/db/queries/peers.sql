-- name: GetPeer :one
SELECT * FROM peers WHERE id = $1;

-- name: GetPeerByPublicKey :one
SELECT * FROM peers WHERE public_key = $1;

-- name: GetPeerByUser :one
SELECT * FROM peers WHERE user_id = $1;

-- name: ListPeersByUser :many
SELECT * FROM peers WHERE user_id = $1 ORDER BY created_at;

-- name: ListEnabledPeers :many
SELECT * FROM peers WHERE enabled = TRUE ORDER BY created_at;

-- name: CreatePeer :one
INSERT INTO peers (user_id, name, public_key, ip_address)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: EnablePeer :exec
UPDATE peers SET enabled = TRUE, updated_at = NOW() WHERE id = $1;

-- name: DisablePeer :exec
UPDATE peers SET enabled = FALSE, updated_at = NOW() WHERE id = $1;

-- name: DeletePeer :exec
DELETE FROM peers WHERE id = $1;

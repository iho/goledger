-- name: CreateHold :one
INSERT INTO holds (id, account_id, amount, status, expires_at, metadata, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetHoldByID :one
SELECT * FROM holds WHERE id = $1;

-- name: GetHoldByIDForUpdate :one
SELECT * FROM holds WHERE id = $1 FOR UPDATE;

-- name: UpdateHoldStatus :exec
UPDATE holds
SET status = $2, updated_at = $3
WHERE id = $1;

-- name: ListHoldsByAccount :many
SELECT * FROM holds
WHERE account_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

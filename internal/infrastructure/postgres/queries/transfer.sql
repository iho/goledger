-- name: CreateTransfer :one
INSERT INTO transfers (id, from_account_id, to_account_id, amount, created_at, event_at, metadata, reversed_transfer_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetTransferByID :one
SELECT * FROM transfers WHERE id = $1;

-- name: ListTransfersByAccount :many
SELECT * FROM transfers
WHERE from_account_id = $1 OR to_account_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountTransfersByAccount :one
SELECT COUNT(*) FROM transfers
WHERE from_account_id = $1 OR to_account_id = $1;

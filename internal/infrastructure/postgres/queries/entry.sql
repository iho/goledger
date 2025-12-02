-- name: CreateEntry :one
INSERT INTO entries (id, account_id, transfer_id, amount, account_previous_balance, account_current_balance, account_version, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetEntriesByTransfer :many
SELECT * FROM entries WHERE transfer_id = $1 ORDER BY created_at;

-- name: GetEntriesByAccount :many
SELECT * FROM entries
WHERE account_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountEntriesByAccount :one
SELECT COUNT(*) FROM entries WHERE account_id = $1;

-- name: GetAccountBalanceAtTime :one
SELECT COALESCE(
    (SELECT account_current_balance FROM entries
     WHERE account_id = $1 AND created_at <= $2
     ORDER BY created_at DESC, id DESC LIMIT 1),
    0
)::NUMERIC AS balance;

-- name: CreateAccount :one
INSERT INTO accounts (id, name, currency, balance, encumbered_balance, version, allow_negative_balance, allow_positive_balance, created_at, updated_at)
VALUES ($1, $2, $3, $4, 0, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetAccountByID :one
SELECT * FROM accounts WHERE id = $1;

-- name: GetAccountByIDForUpdate :one
SELECT * FROM accounts WHERE id = $1 FOR UPDATE;

-- name: GetAccountsByIDsForUpdate :many
SELECT * FROM accounts WHERE id = ANY($1::text[]) ORDER BY id FOR UPDATE;

-- name: UpdateAccountBalance :exec
UPDATE accounts
SET balance = $2, version = version + 1, updated_at = $3
WHERE id = $1;

-- name: UpdateAccountEncumbered :exec
UPDATE accounts
SET encumbered_balance = $2, version = version + 1, updated_at = $3
WHERE id = $1;

-- name: ListAccounts :many
SELECT * FROM accounts ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: CountAccounts :one
SELECT COUNT(*) FROM accounts;

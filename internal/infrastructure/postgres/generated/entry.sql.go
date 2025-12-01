
package generated

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const countEntriesByAccount = `-- name: CountEntriesByAccount :one
SELECT COUNT(*) FROM entries WHERE account_id = $1
`

func (q *Queries) CountEntriesByAccount(ctx context.Context, accountID string) (int64, error) {
	row := q.db.QueryRow(ctx, countEntriesByAccount, accountID)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const createEntry = `-- name: CreateEntry :one
INSERT INTO entries (id, account_id, transfer_id, amount, account_previous_balance, account_current_balance, account_version, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, account_id, transfer_id, amount, account_previous_balance, account_current_balance, account_version, created_at
`

type CreateEntryParams struct {
	ID                     string             `json:"id"`
	AccountID              string             `json:"account_id"`
	TransferID             string             `json:"transfer_id"`
	Amount                 pgtype.Numeric     `json:"amount"`
	AccountPreviousBalance pgtype.Numeric     `json:"account_previous_balance"`
	AccountCurrentBalance  pgtype.Numeric     `json:"account_current_balance"`
	AccountVersion         int64              `json:"account_version"`
	CreatedAt              pgtype.Timestamptz `json:"created_at"`
}

func (q *Queries) CreateEntry(ctx context.Context, arg CreateEntryParams) (Entry, error) {
	row := q.db.QueryRow(ctx, createEntry,
		arg.ID,
		arg.AccountID,
		arg.TransferID,
		arg.Amount,
		arg.AccountPreviousBalance,
		arg.AccountCurrentBalance,
		arg.AccountVersion,
		arg.CreatedAt,
	)
	var i Entry
	err := row.Scan(
		&i.ID,
		&i.AccountID,
		&i.TransferID,
		&i.Amount,
		&i.AccountPreviousBalance,
		&i.AccountCurrentBalance,
		&i.AccountVersion,
		&i.CreatedAt,
	)
	return i, err
}

const getAccountBalanceAtTime = `-- name: GetAccountBalanceAtTime :one
SELECT COALESCE(
    (SELECT account_current_balance FROM entries
     WHERE account_id = $1 AND created_at <= $2
     ORDER BY created_at DESC, id DESC LIMIT 1),
    0
)::NUMERIC AS balance
`

type GetAccountBalanceAtTimeParams struct {
	AccountID string             `json:"account_id"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
}

func (q *Queries) GetAccountBalanceAtTime(ctx context.Context, arg GetAccountBalanceAtTimeParams) (pgtype.Numeric, error) {
	row := q.db.QueryRow(ctx, getAccountBalanceAtTime, arg.AccountID, arg.CreatedAt)
	var balance pgtype.Numeric
	err := row.Scan(&balance)
	return balance, err
}

const getEntriesByAccount = `-- name: GetEntriesByAccount :many
SELECT id, account_id, transfer_id, amount, account_previous_balance, account_current_balance, account_version, created_at FROM entries
WHERE account_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3
`

type GetEntriesByAccountParams struct {
	AccountID string `json:"account_id"`
	Limit     int32  `json:"limit"`
	Offset    int32  `json:"offset"`
}

func (q *Queries) GetEntriesByAccount(ctx context.Context, arg GetEntriesByAccountParams) ([]Entry, error) {
	rows, err := q.db.Query(ctx, getEntriesByAccount, arg.AccountID, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Entry{}
	for rows.Next() {
		var i Entry
		if err := rows.Scan(
			&i.ID,
			&i.AccountID,
			&i.TransferID,
			&i.Amount,
			&i.AccountPreviousBalance,
			&i.AccountCurrentBalance,
			&i.AccountVersion,
			&i.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getEntriesByTransfer = `-- name: GetEntriesByTransfer :many
SELECT id, account_id, transfer_id, amount, account_previous_balance, account_current_balance, account_version, created_at FROM entries WHERE transfer_id = $1 ORDER BY created_at
`

func (q *Queries) GetEntriesByTransfer(ctx context.Context, transferID string) ([]Entry, error) {
	rows, err := q.db.Query(ctx, getEntriesByTransfer, transferID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Entry{}
	for rows.Next() {
		var i Entry
		if err := rows.Scan(
			&i.ID,
			&i.AccountID,
			&i.TransferID,
			&i.Amount,
			&i.AccountPreviousBalance,
			&i.AccountCurrentBalance,
			&i.AccountVersion,
			&i.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

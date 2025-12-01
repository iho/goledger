
package generated

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const countAccounts = `-- name: CountAccounts :one
SELECT COUNT(*) FROM accounts
`

func (q *Queries) CountAccounts(ctx context.Context) (int64, error) {
	row := q.db.QueryRow(ctx, countAccounts)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const createAccount = `-- name: CreateAccount :one
INSERT INTO accounts (id, name, currency, balance, version, allow_negative_balance, allow_positive_balance, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, name, currency, balance, version, allow_negative_balance, allow_positive_balance, created_at, updated_at
`

type CreateAccountParams struct {
	ID                   string             `json:"id"`
	Name                 string             `json:"name"`
	Currency             string             `json:"currency"`
	Balance              pgtype.Numeric     `json:"balance"`
	Version              int64              `json:"version"`
	AllowNegativeBalance bool               `json:"allow_negative_balance"`
	AllowPositiveBalance bool               `json:"allow_positive_balance"`
	CreatedAt            pgtype.Timestamptz `json:"created_at"`
	UpdatedAt            pgtype.Timestamptz `json:"updated_at"`
}

func (q *Queries) CreateAccount(ctx context.Context, arg CreateAccountParams) (Account, error) {
	row := q.db.QueryRow(ctx, createAccount,
		arg.ID,
		arg.Name,
		arg.Currency,
		arg.Balance,
		arg.Version,
		arg.AllowNegativeBalance,
		arg.AllowPositiveBalance,
		arg.CreatedAt,
		arg.UpdatedAt,
	)
	var i Account
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Currency,
		&i.Balance,
		&i.Version,
		&i.AllowNegativeBalance,
		&i.AllowPositiveBalance,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getAccountByID = `-- name: GetAccountByID :one
SELECT id, name, currency, balance, version, allow_negative_balance, allow_positive_balance, created_at, updated_at FROM accounts WHERE id = $1
`

func (q *Queries) GetAccountByID(ctx context.Context, id string) (Account, error) {
	row := q.db.QueryRow(ctx, getAccountByID, id)
	var i Account
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Currency,
		&i.Balance,
		&i.Version,
		&i.AllowNegativeBalance,
		&i.AllowPositiveBalance,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getAccountByIDForUpdate = `-- name: GetAccountByIDForUpdate :one
SELECT id, name, currency, balance, version, allow_negative_balance, allow_positive_balance, created_at, updated_at FROM accounts WHERE id = $1 FOR UPDATE
`

func (q *Queries) GetAccountByIDForUpdate(ctx context.Context, id string) (Account, error) {
	row := q.db.QueryRow(ctx, getAccountByIDForUpdate, id)
	var i Account
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Currency,
		&i.Balance,
		&i.Version,
		&i.AllowNegativeBalance,
		&i.AllowPositiveBalance,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getAccountsByIDsForUpdate = `-- name: GetAccountsByIDsForUpdate :many
SELECT id, name, currency, balance, version, allow_negative_balance, allow_positive_balance, created_at, updated_at FROM accounts WHERE id = ANY($1::text[]) ORDER BY id FOR UPDATE
`

func (q *Queries) GetAccountsByIDsForUpdate(ctx context.Context, dollar_1 []string) ([]Account, error) {
	rows, err := q.db.Query(ctx, getAccountsByIDsForUpdate, dollar_1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Account{}
	for rows.Next() {
		var i Account
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Currency,
			&i.Balance,
			&i.Version,
			&i.AllowNegativeBalance,
			&i.AllowPositiveBalance,
			&i.CreatedAt,
			&i.UpdatedAt,
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

const listAccounts = `-- name: ListAccounts :many
SELECT id, name, currency, balance, version, allow_negative_balance, allow_positive_balance, created_at, updated_at FROM accounts ORDER BY created_at DESC LIMIT $1 OFFSET $2
`

type ListAccountsParams struct {
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}

func (q *Queries) ListAccounts(ctx context.Context, arg ListAccountsParams) ([]Account, error) {
	rows, err := q.db.Query(ctx, listAccounts, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Account{}
	for rows.Next() {
		var i Account
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Currency,
			&i.Balance,
			&i.Version,
			&i.AllowNegativeBalance,
			&i.AllowPositiveBalance,
			&i.CreatedAt,
			&i.UpdatedAt,
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

const updateAccountBalance = `-- name: UpdateAccountBalance :exec
UPDATE accounts
SET balance = $2, version = version + 1, updated_at = $3
WHERE id = $1
`

type UpdateAccountBalanceParams struct {
	ID        string             `json:"id"`
	Balance   pgtype.Numeric     `json:"balance"`
	UpdatedAt pgtype.Timestamptz `json:"updated_at"`
}

func (q *Queries) UpdateAccountBalance(ctx context.Context, arg UpdateAccountBalanceParams) error {
	_, err := q.db.Exec(ctx, updateAccountBalance, arg.ID, arg.Balance, arg.UpdatedAt)
	return err
}

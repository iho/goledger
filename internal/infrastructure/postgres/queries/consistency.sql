-- name: CheckLedgerConsistency :one
SELECT
    (SELECT COALESCE(SUM(balance), 0)::NUMERIC FROM accounts) as total_account_balance,
    (SELECT COALESCE(SUM(amount), 0)::NUMERIC FROM entries) as total_entry_amount;

-- name: CheckLedgerConsistencyByCurrency :many
-- Same invariant as CheckLedgerConsistency, but grouped by currency so
-- offsetting errors in different currencies don't cancel each other out in
-- one global sum.
WITH balance_by_currency AS (
    SELECT currency, COALESCE(SUM(balance), 0)::NUMERIC AS total_balance
    FROM accounts
    GROUP BY currency
),
entries_by_currency AS (
    SELECT a.currency, COALESCE(SUM(e.amount), 0)::NUMERIC AS total_amount
    FROM entries e
    JOIN accounts a ON a.id = e.account_id
    GROUP BY a.currency
)
SELECT
    b.currency AS currency,
    b.total_balance AS total_account_balance,
    COALESCE(e.total_amount, 0)::NUMERIC AS total_entry_amount
FROM balance_by_currency b
LEFT JOIN entries_by_currency e ON e.currency = b.currency
ORDER BY b.currency;

-- name: SumEntryAmountsByAccount :one
-- Total of all entries for an account. Because an account's balance always
-- starts at zero and only ever changes via entries, this should equal the
-- account's current recorded balance.
SELECT COALESCE(SUM(amount), 0)::NUMERIC AS total_amount
FROM entries
WHERE account_id = $1;

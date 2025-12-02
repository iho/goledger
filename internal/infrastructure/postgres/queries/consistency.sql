-- name: CheckLedgerConsistency :one
SELECT
    (SELECT COALESCE(SUM(balance), 0)::NUMERIC FROM accounts) as total_account_balance,
    (SELECT COALESCE(SUM(amount), 0)::NUMERIC FROM entries) as total_entry_amount;

ALTER TABLE accounts DROP CONSTRAINT IF EXISTS chk_accounts_positive_balance;
ALTER TABLE accounts DROP CONSTRAINT IF EXISTS chk_accounts_available_balance;
ALTER TABLE accounts DROP CONSTRAINT IF EXISTS chk_accounts_encumbered_non_negative;

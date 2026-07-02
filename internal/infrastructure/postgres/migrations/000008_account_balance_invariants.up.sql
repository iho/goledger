-- Enforce at the database level the balance invariants the application
-- already assumes: an account's available balance (balance minus what is
-- encumbered by active holds) can never go negative unless the account
-- explicitly allows it, and balance can never go positive unless the
-- account explicitly allows it. encumbered_balance itself can never be
-- negative.
ALTER TABLE accounts ADD CONSTRAINT chk_accounts_encumbered_non_negative CHECK (encumbered_balance >= 0);
ALTER TABLE accounts ADD CONSTRAINT chk_accounts_available_balance CHECK (allow_negative_balance OR balance - encumbered_balance >= 0);
ALTER TABLE accounts ADD CONSTRAINT chk_accounts_positive_balance CHECK (allow_positive_balance OR balance <= 0);

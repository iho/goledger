ALTER TABLE accounts ADD COLUMN encumbered_balance NUMERIC NOT NULL DEFAULT 0;

CREATE TABLE holds (
    id TEXT PRIMARY KEY,
    account_id TEXT NOT NULL REFERENCES accounts(id),
    amount NUMERIC NOT NULL,
    status TEXT NOT NULL, -- 'active', 'voided', 'captured'
    expires_at TIMESTAMPTZ,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CHECK (amount > 0)
);

CREATE INDEX idx_holds_account ON holds(account_id);
CREATE INDEX idx_holds_status ON holds(status);

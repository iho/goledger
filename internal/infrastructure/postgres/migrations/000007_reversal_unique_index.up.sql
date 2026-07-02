-- Enforce that a transfer can be reversed at most once: only one reversal
-- transfer may reference the same original transfer.
DROP INDEX IF EXISTS idx_transfers_reversed_transfer_id;
CREATE UNIQUE INDEX idx_transfers_reversed_transfer_id ON transfers(reversed_transfer_id) WHERE reversed_transfer_id IS NOT NULL;

-- Remove index
DROP INDEX IF EXISTS idx_transfers_reversed_transfer_id;

-- Remove reversed_transfer_id column
ALTER TABLE transfers DROP COLUMN IF EXISTS reversed_transfer_id;

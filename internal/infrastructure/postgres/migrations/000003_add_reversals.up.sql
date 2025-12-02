-- Add reversed_transfer_id column to transfers table
ALTER TABLE transfers ADD COLUMN reversed_transfer_id TEXT REFERENCES transfers(id);

-- Create index for faster lookup of reversals
CREATE INDEX idx_transfers_reversed_transfer_id ON transfers(reversed_transfer_id);

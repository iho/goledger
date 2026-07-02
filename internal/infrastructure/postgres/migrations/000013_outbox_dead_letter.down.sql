DROP INDEX IF EXISTS idx_outbox_events_dead_lettered;
ALTER TABLE outbox_events DROP COLUMN IF EXISTS dead_lettered_at;
ALTER TABLE outbox_events DROP COLUMN IF EXISTS last_error;
ALTER TABLE outbox_events DROP COLUMN IF EXISTS attempts;

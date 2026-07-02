DROP INDEX IF EXISTS idx_outbox_events_aggregate_sequence;
ALTER TABLE outbox_events DROP COLUMN IF EXISTS aggregate_sequence;
ALTER TABLE outbox_events DROP COLUMN IF EXISTS event_version;

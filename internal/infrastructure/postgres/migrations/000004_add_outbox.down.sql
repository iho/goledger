-- Remove indexes
DROP INDEX IF EXISTS idx_outbox_events_aggregate;
DROP INDEX IF EXISTS idx_outbox_events_unpublished;

-- Remove outbox_events table
DROP TABLE IF EXISTS outbox_events;

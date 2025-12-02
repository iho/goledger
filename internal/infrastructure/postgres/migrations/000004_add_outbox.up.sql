-- Create outbox_events table for transactional outbox pattern
CREATE TABLE outbox_events (
    id TEXT PRIMARY KEY,
    aggregate_id TEXT NOT NULL,
    aggregate_type TEXT NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    published_at TIMESTAMPTZ,
    published BOOLEAN NOT NULL DEFAULT FALSE
);

-- Index for efficient polling of unpublished events
CREATE INDEX idx_outbox_events_unpublished ON outbox_events(published, created_at) WHERE NOT published;

-- Index for querying by aggregate
CREATE INDEX idx_outbox_events_aggregate ON outbox_events(aggregate_type, aggregate_id);

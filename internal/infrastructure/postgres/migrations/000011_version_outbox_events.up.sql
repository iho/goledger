-- Let downstream consumers evolve payload formats and detect gaps/out-of-order
-- delivery: event_version tags the payload schema, aggregate_sequence is a
-- per-aggregate monotonically increasing counter (1, 2, 3... within one
-- aggregate_type+aggregate_id).
ALTER TABLE outbox_events ADD COLUMN event_version INT NOT NULL DEFAULT 1;
ALTER TABLE outbox_events ADD COLUMN aggregate_sequence BIGINT NOT NULL DEFAULT 1;

-- Backfill sequence numbers for any existing rows in creation order per aggregate.
WITH numbered AS (
    SELECT id, ROW_NUMBER() OVER (
        PARTITION BY aggregate_type, aggregate_id ORDER BY created_at, id
    ) AS seq
    FROM outbox_events
)
UPDATE outbox_events
SET aggregate_sequence = numbered.seq
FROM numbered
WHERE outbox_events.id = numbered.id;

CREATE UNIQUE INDEX idx_outbox_events_aggregate_sequence
    ON outbox_events(aggregate_type, aggregate_id, aggregate_sequence);

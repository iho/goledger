-- name: CreateOutboxEvent :one
-- aggregate_sequence is computed as the next number for this aggregate,
-- relying on the caller already holding a row lock on the aggregate (the
-- account/hold FOR UPDATE taken earlier in the same transaction) to
-- serialize concurrent writers - see idx_outbox_events_aggregate_sequence
-- for the invariant this must uphold.
INSERT INTO outbox_events (id, aggregate_id, aggregate_type, event_type, event_version, aggregate_sequence, payload, created_at, published)
VALUES (
    $1, $2, $3, $4, $5,
    COALESCE((SELECT MAX(aggregate_sequence) FROM outbox_events WHERE aggregate_type = $3 AND aggregate_id = $2), 0) + 1,
    $6, $7, $8
)
RETURNING *;

-- name: GetUnpublishedEvents :many
-- Excludes dead-lettered events so one poison message can't block the
-- whole queue behind it; see RecordOutboxFailure/MarkOutboxDeadLettered.
SELECT * FROM outbox_events
WHERE published = FALSE AND dead_lettered_at IS NULL
ORDER BY created_at ASC
LIMIT $1;

-- name: MarkEventPublished :exec
UPDATE outbox_events
SET published = TRUE, published_at = $2
WHERE id = $1;

-- name: RecordOutboxFailure :one
UPDATE outbox_events
SET attempts = attempts + 1, last_error = $2
WHERE id = $1
RETURNING attempts;

-- name: MarkOutboxDeadLettered :exec
UPDATE outbox_events
SET dead_lettered_at = $2
WHERE id = $1;

-- name: GetDeadLetteredEvents :many
SELECT * FROM outbox_events
WHERE dead_lettered_at IS NOT NULL
ORDER BY dead_lettered_at DESC
LIMIT $1 OFFSET $2;

-- name: GetEventsByAggregate :many
SELECT * FROM outbox_events
WHERE aggregate_type = $1 AND aggregate_id = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: DeletePublishedEvents :exec
DELETE FROM outbox_events
WHERE published = TRUE AND published_at < $1;

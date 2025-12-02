-- name: CreateOutboxEvent :one
INSERT INTO outbox_events (id, aggregate_id, aggregate_type, event_type, payload, created_at, published)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetUnpublishedEvents :many
SELECT * FROM outbox_events
WHERE published = FALSE
ORDER BY created_at ASC
LIMIT $1;

-- name: MarkEventPublished :exec
UPDATE outbox_events
SET published = TRUE, published_at = $2
WHERE id = $1;

-- name: GetEventsByAggregate :many
SELECT * FROM outbox_events
WHERE aggregate_type = $1 AND aggregate_id = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: DeletePublishedEvents :exec
DELETE FROM outbox_events
WHERE published = TRUE AND published_at < $1;

package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/infrastructure/postgres/generated"
	"github.com/iho/goledger/internal/usecase"
)

// OutboxRepository implements usecase.OutboxRepository.
type OutboxRepository struct {
	pool    *pgxpool.Pool
	queries *generated.Queries
}

// NewOutboxRepository creates a new OutboxRepository.
func NewOutboxRepository(pool *pgxpool.Pool) *OutboxRepository {
	return &OutboxRepository{
		pool:    pool,
		queries: generated.New(pool),
	}
}

// Create creates a new outbox event within a transaction.
func (r *OutboxRepository) Create(ctx context.Context, tx usecase.Transaction, event *domain.OutboxEvent) error {
	pgxTx := tx.(*Tx).PgxTx()
	queries := generated.New(pgxTx)

	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return err
	}

	_, err = queries.CreateOutboxEvent(ctx, generated.CreateOutboxEventParams{
		ID:            event.ID,
		AggregateID:   event.AggregateID,
		AggregateType: event.AggregateType,
		EventType:     event.EventType,
		Payload:       payload,
		CreatedAt:     timeToPgTimestamptz(event.CreatedAt),
		Published:     event.Published,
	})

	return err
}

// GetUnpublished retrieves unpublished events.
func (r *OutboxRepository) GetUnpublished(ctx context.Context, limit int) ([]*domain.OutboxEvent, error) {
	rows, err := r.queries.GetUnpublishedEvents(ctx, int32(limit))
	if err != nil {
		return nil, err
	}

	events := make([]*domain.OutboxEvent, 0, len(rows))
	for _, row := range rows {
		events = append(events, rowToOutboxEvent(row))
	}

	return events, nil
}

// MarkPublished marks an event as published.
func (r *OutboxRepository) MarkPublished(ctx context.Context, id string, publishedAt time.Time) error {
	return r.queries.MarkEventPublished(ctx, generated.MarkEventPublishedParams{
		ID:          id,
		PublishedAt: timeToPgTimestamptz(publishedAt),
	})
}

// GetByAggregate retrieves events for a specific aggregate.
func (r *OutboxRepository) GetByAggregate(ctx context.Context, aggregateType, aggregateID string, limit, offset int) ([]*domain.OutboxEvent, error) {
	rows, err := r.queries.GetEventsByAggregate(ctx, generated.GetEventsByAggregateParams{
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		Limit:         int32(limit),
		Offset:        int32(offset),
	})
	if err != nil {
		return nil, err
	}

	events := make([]*domain.OutboxEvent, 0, len(rows))
	for _, row := range rows {
		events = append(events, rowToOutboxEvent(row))
	}

	return events, nil
}

// DeletePublished deletes published events older than the given time.
func (r *OutboxRepository) DeletePublished(ctx context.Context, before time.Time) error {
	return r.queries.DeletePublishedEvents(ctx, timeToPgTimestamptz(before))
}

func rowToOutboxEvent(row generated.OutboxEvent) *domain.OutboxEvent {
	var payload map[string]any
	if row.Payload != nil {
		_ = json.Unmarshal(row.Payload, &payload)
	}

	var publishedAt *time.Time
	if row.PublishedAt.Valid {
		t := row.PublishedAt.Time
		publishedAt = &t
	}

	return &domain.OutboxEvent{
		ID:            row.ID,
		AggregateID:   row.AggregateID,
		AggregateType: row.AggregateType,
		EventType:     row.EventType,
		Payload:       payload,
		CreatedAt:     row.CreatedAt.Time,
		PublishedAt:   publishedAt,
		Published:     row.Published,
	}
}

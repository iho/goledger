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

	eventVersion := event.EventVersion
	if eventVersion == 0 {
		eventVersion = 1
	}

	row, err := queries.CreateOutboxEvent(ctx, generated.CreateOutboxEventParams{
		ID:            event.ID,
		AggregateID:   event.AggregateID,
		AggregateType: event.AggregateType,
		EventType:     event.EventType,
		EventVersion:  eventVersion,
		Payload:       payload,
		CreatedAt:     timeToPgTimestamptz(event.CreatedAt),
		Published:     event.Published,
	})
	if err != nil {
		return err
	}

	event.EventVersion = row.EventVersion
	event.AggregateSequence = row.AggregateSequence

	return nil
}

// GetUnpublished retrieves unpublished events.
func (r *OutboxRepository) GetUnpublished(ctx context.Context, limit int) ([]*domain.OutboxEvent, error) {
	rows, err := r.queries.GetUnpublishedEvents(ctx, toInt32(limit))
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
		Limit:         toInt32(limit),
		Offset:        toInt32(offset),
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

// RecordFailure increments the event's attempt counter and stores the error.
func (r *OutboxRepository) RecordFailure(ctx context.Context, id, lastError string) (int, error) {
	attempts, err := r.queries.RecordOutboxFailure(ctx, generated.RecordOutboxFailureParams{
		ID:        id,
		LastError: &lastError,
	})
	if err != nil {
		return 0, err
	}

	return int(attempts), nil
}

// MarkDeadLettered stops the publisher from retrying this event.
func (r *OutboxRepository) MarkDeadLettered(ctx context.Context, id string, at time.Time) error {
	return r.queries.MarkOutboxDeadLettered(ctx, generated.MarkOutboxDeadLetteredParams{
		ID:             id,
		DeadLetteredAt: timeToPgTimestamptz(at),
	})
}

// GetDeadLettered lists dead-lettered events for operator inspection.
func (r *OutboxRepository) GetDeadLettered(ctx context.Context, limit, offset int) ([]*domain.OutboxEvent, error) {
	rows, err := r.queries.GetDeadLetteredEvents(ctx, generated.GetDeadLetteredEventsParams{
		Limit:  toInt32(limit),
		Offset: toInt32(offset),
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

	var deadLetteredAt *time.Time
	if row.DeadLetteredAt.Valid {
		t := row.DeadLetteredAt.Time
		deadLetteredAt = &t
	}

	var lastError string
	if row.LastError != nil {
		lastError = *row.LastError
	}

	return &domain.OutboxEvent{
		ID:                row.ID,
		AggregateID:       row.AggregateID,
		AggregateType:     row.AggregateType,
		EventType:         row.EventType,
		Payload:           payload,
		CreatedAt:         row.CreatedAt.Time,
		PublishedAt:       publishedAt,
		Published:         row.Published,
		EventVersion:      row.EventVersion,
		AggregateSequence: row.AggregateSequence,
		Attempts:          int(row.Attempts),
		LastError:         lastError,
		DeadLetteredAt:    deadLetteredAt,
	}
}

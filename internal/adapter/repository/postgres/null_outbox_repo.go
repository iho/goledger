package postgres

import (
	"context"
	"time"

	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/usecase"
)

// NullOutboxRepository is a no-op implementation for tests.
type NullOutboxRepository struct{}

// NewNullOutboxRepository creates a new NullOutboxRepository.
func NewNullOutboxRepository() *NullOutboxRepository {
	return &NullOutboxRepository{}
}

func (r *NullOutboxRepository) Create(ctx context.Context, tx usecase.Transaction, event *domain.OutboxEvent) error {
	return nil
}

func (r *NullOutboxRepository) GetUnpublished(ctx context.Context, limit int) ([]*domain.OutboxEvent, error) {
	return nil, nil
}

func (r *NullOutboxRepository) MarkPublished(ctx context.Context, id string, publishedAt time.Time) error {
	return nil
}

func (r *NullOutboxRepository) GetByAggregate(ctx context.Context, aggregateType, aggregateID string, limit, offset int) ([]*domain.OutboxEvent, error) {
	return nil, nil
}

func (r *NullOutboxRepository) DeletePublished(ctx context.Context, before time.Time) error {
	return nil
}

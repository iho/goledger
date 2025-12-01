package eventpublisher

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/usecase"
)

// EventPublisher handles publishing events from the outbox.
type EventPublisher struct {
	outboxRepo usecase.OutboxRepository
	publisher  Publisher
	logger     *slog.Logger
	batchSize  int
	interval   time.Duration
}

// Publisher defines the interface for publishing events to external systems.
type Publisher interface {
	Publish(ctx context.Context, event *domain.OutboxEvent) error
}

// Config for EventPublisher.
type Config struct {
	OutboxRepo usecase.OutboxRepository
	Publisher  Publisher
	Logger     *slog.Logger
	BatchSize  int           // Number of events to fetch per batch
	Interval   time.Duration // Polling interval
}

// NewEventPublisher creates a new EventPublisher.
func NewEventPublisher(cfg Config) *EventPublisher {
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 100
	}
	if cfg.Interval == 0 {
		cfg.Interval = 5 * time.Second
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	return &EventPublisher{
		outboxRepo: cfg.OutboxRepo,
		publisher:  cfg.Publisher,
		logger:     cfg.Logger,
		batchSize:  cfg.BatchSize,
		interval:   cfg.Interval,
	}
}

// Start begins the event publishing worker.
// It runs continuously until the context is cancelled.
func (ep *EventPublisher) Start(ctx context.Context) error {
	ep.logger.Info("event publisher started",
		slog.Int("batch_size", ep.batchSize),
		slog.Duration("interval", ep.interval))

	ticker := time.NewTicker(ep.interval)
	defer ticker.Stop()

	// Process immediately on start
	if err := ep.processEvents(ctx); err != nil {
		ep.logger.Error("error processing events on start", slog.String("error", err.Error()))
	}

	for {
		select {
		case <-ctx.Done():
			ep.logger.Info("event publisher shutting down")
			return ctx.Err()
		case <-ticker.C:
			if err := ep.processEvents(ctx); err != nil {
				ep.logger.Error("error processing events", slog.String("error", err.Error()))
			}
		}
	}
}

// processEvents fetches and publishes a batch of unpublished events.
func (ep *EventPublisher) processEvents(ctx context.Context) error {
	events, err := ep.outboxRepo.GetUnpublished(ctx, ep.batchSize)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		return nil
	}

	ep.logger.Info("processing events", slog.Int("count", len(events)))

	for _, event := range events {
		if err := ep.publishEvent(ctx, event); err != nil {
			ep.logger.Error("failed to publish event",
				slog.String("event_id", event.ID),
				slog.String("event_type", event.EventType),
				slog.String("error", err.Error()))
			// Continue processing other events even if one fails
			continue
		}

		// Mark as published
		if err := ep.outboxRepo.MarkPublished(ctx, event.ID, time.Now()); err != nil {
			ep.logger.Error("failed to mark event as published",
				slog.String("event_id", event.ID),
				slog.String("error", err.Error()))
			// Don't continue - we don't want to re-publish this event
		}
	}

	return nil
}

// publishEvent publishes a single event.
func (ep *EventPublisher) publishEvent(ctx context.Context, event *domain.OutboxEvent) error {
	ep.logger.Debug("publishing event",
		slog.String("event_id", event.ID),
		slog.String("event_type", event.EventType),
		slog.String("aggregate_type", event.AggregateType),
		slog.String("aggregate_id", event.AggregateID))

	if err := ep.publisher.Publish(ctx, event); err != nil {
		return err
	}

	ep.logger.Info("event published",
		slog.String("event_id", event.ID),
		slog.String("event_type", event.EventType))

	return nil
}

// LogPublisher is a simple publisher that logs events.
type LogPublisher struct {
	logger *slog.Logger
}

// NewLogPublisher creates a new LogPublisher.
func NewLogPublisher(logger *slog.Logger) *LogPublisher {
	if logger == nil {
		logger = slog.Default()
	}
	return &LogPublisher{logger: logger}
}

// Publish logs the event.
func (p *LogPublisher) Publish(ctx context.Context, event *domain.OutboxEvent) error {
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return err
	}

	p.logger.Info("EVENT PUBLISHED",
		slog.String("event_id", event.ID),
		slog.String("event_type", event.EventType),
		slog.String("aggregate_type", event.AggregateType),
		slog.String("aggregate_id", event.AggregateID),
		slog.String("payload", string(payload)))

	return nil
}

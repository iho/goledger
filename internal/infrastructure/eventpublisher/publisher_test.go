package eventpublisher

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/usecase"
)

func TestProcessEventsPublishesAndMarks(t *testing.T) {
	repo := &stubOutboxRepo{
		events: []*domain.OutboxEvent{{ID: "evt-1", EventType: "type"}},
	}
	pub := &stubPublisher{}
	ep := newTestPublisher(repo, pub)

	if err := ep.processEvents(context.Background()); err != nil {
		t.Fatalf("processEvents failed: %v", err)
	}

	if len(pub.published) != 1 {
		t.Fatalf("expected one published event, got %d", len(pub.published))
	}
	if len(repo.marked) != 1 || repo.marked[0] != "evt-1" {
		t.Fatalf("expected event to be marked published, got %#v", repo.marked)
	}
}

func TestProcessEventsContinuesOnPublishError(t *testing.T) {
	repo := &stubOutboxRepo{
		events: []*domain.OutboxEvent{
			{ID: "evt-1", EventType: "type"},
			{ID: "evt-2", EventType: "type"},
		},
	}
	pub := &stubPublisher{
		errorsByID: map[string]error{"evt-1": errors.New("fail")},
	}
	ep := newTestPublisher(repo, pub)

	if err := ep.processEvents(context.Background()); err != nil {
		t.Fatalf("processEvents returned error: %v", err)
	}

	if len(pub.published) != 1 || pub.published[0].ID != "evt-2" {
		t.Fatalf("expected only evt-2 to be published, got %#v", pub.published)
	}
	if len(repo.marked) != 1 || repo.marked[0] != "evt-2" {
		t.Fatalf("expected only evt-2 to be marked, got %#v", repo.marked)
	}
}

func TestProcessEventsDeadLettersAfterMaxAttempts(t *testing.T) {
	repo := &stubOutboxRepo{
		events: []*domain.OutboxEvent{{ID: "evt-1", EventType: "type"}},
	}
	pub := &stubPublisher{
		errorsByID: map[string]error{"evt-1": errors.New("permanently broken")},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	ep := NewEventPublisher(Config{
		OutboxRepo:  repo,
		Publisher:   pub,
		Logger:      logger,
		BatchSize:   10,
		Interval:    5 * time.Millisecond,
		MaxAttempts: 3,
	})

	for range 2 {
		if err := ep.processEvents(context.Background()); err != nil {
			t.Fatalf("processEvents failed: %v", err)
		}
		if len(repo.deadLettered) != 0 {
			t.Fatalf("expected no dead-lettering before max attempts, got %#v", repo.deadLettered)
		}
	}

	if err := ep.processEvents(context.Background()); err != nil {
		t.Fatalf("processEvents failed: %v", err)
	}

	if len(repo.deadLettered) != 1 || repo.deadLettered[0] != "evt-1" {
		t.Fatalf("expected evt-1 to be dead-lettered after 3 attempts, got %#v", repo.deadLettered)
	}
	if repo.failures["evt-1"] != 3 {
		t.Fatalf("expected 3 recorded failures, got %d", repo.failures["evt-1"])
	}
}

func TestStartStopsOnContextCancellation(t *testing.T) {
	repo := &stubOutboxRepo{}
	pub := &stubPublisher{}
	ep := newTestPublisher(repo, pub)
	ep.interval = 10 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- ep.Start(ctx)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context canceled, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("publisher did not stop after cancel")
	}
}

func newTestPublisher(repo *stubOutboxRepo, pub *stubPublisher) *EventPublisher {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	return NewEventPublisher(Config{
		OutboxRepo: repo,
		Publisher:  pub,
		Logger:     logger,
		BatchSize:  10,
		Interval:   5 * time.Millisecond,
	})
}

type stubOutboxRepo struct {
	events       []*domain.OutboxEvent
	marked       []string
	failures     map[string]int
	deadLettered []string
}

func (s *stubOutboxRepo) Create(ctx context.Context, tx usecase.Transaction, event *domain.OutboxEvent) error {
	return nil
}

func (s *stubOutboxRepo) GetUnpublished(ctx context.Context, limit int) ([]*domain.OutboxEvent, error) {
	if len(s.events) <= limit {
		return append([]*domain.OutboxEvent(nil), s.events...), nil
	}
	return append([]*domain.OutboxEvent(nil), s.events[:limit]...), nil
}

func (s *stubOutboxRepo) MarkPublished(ctx context.Context, id string, publishedAt time.Time) error {
	s.marked = append(s.marked, id)
	return nil
}

func (s *stubOutboxRepo) GetByAggregate(ctx context.Context, aggregateType, aggregateID string, limit, offset int) ([]*domain.OutboxEvent, error) {
	return nil, nil
}

func (s *stubOutboxRepo) DeletePublished(ctx context.Context, before time.Time) error {
	return nil
}

func (s *stubOutboxRepo) RecordFailure(ctx context.Context, id, lastError string) (int, error) {
	if s.failures == nil {
		s.failures = make(map[string]int)
	}
	s.failures[id]++
	return s.failures[id], nil
}

func (s *stubOutboxRepo) MarkDeadLettered(ctx context.Context, id string, at time.Time) error {
	s.deadLettered = append(s.deadLettered, id)
	return nil
}

func (s *stubOutboxRepo) GetDeadLettered(ctx context.Context, limit, offset int) ([]*domain.OutboxEvent, error) {
	return nil, nil
}

type stubPublisher struct {
	published  []*domain.OutboxEvent
	errorsByID map[string]error
}

func (s *stubPublisher) Publish(ctx context.Context, event *domain.OutboxEvent) error {
	if err := s.errorsByID[event.ID]; err != nil {
		return err
	}
	s.published = append(s.published, event)
	return nil
}

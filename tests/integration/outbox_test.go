package integration

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/iho/goledger/internal/adapter/repository/postgres"
	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/infrastructure/eventpublisher"
	"github.com/iho/goledger/internal/usecase"
	"github.com/iho/goledger/tests/testutil"
)

func TestOutboxEventCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	testDB := testutil.NewTestDB(t)
	defer testDB.Cleanup()

	pool := testDB.Pool
	accountRepo := postgres.NewAccountRepository(pool)
	transferRepo := postgres.NewTransferRepository(pool)
	entryRepo := postgres.NewEntryRepository(pool)
	outboxRepo := postgres.NewOutboxRepository(pool)
	txManager := postgres.NewTxManager(pool)
	idGen := postgres.NewULIDGenerator()
	retrier := postgres.NewRetrier()

	transferUC := usecase.NewTransferUseCase(txManager, accountRepo, transferRepo, entryRepo, outboxRepo, nil, idGen, nil).WithRetrier(retrier)

	// Create accounts with balance
	acc1 := testDB.CreateTestAccountWithBalance(ctx, "acc1", "USD", decimal.NewFromInt(1000), false, true)
	acc2 := testDB.CreateTestAccount(ctx, "acc2", "USD", false, true)

	// Create a transfer
	transfer, err := transferUC.CreateTransfer(ctx, usecase.CreateTransferInput{
		FromAccountID: acc1.ID,
		ToAccountID:   acc2.ID,
		Amount:        decimal.NewFromInt(100),
	})
	if err != nil {
		t.Fatalf("failed to create transfer: %v", err)
	}

	// Verify outbox event was created
	events, err := outboxRepo.GetUnpublished(ctx, 10)
	if err != nil {
		t.Fatalf("failed to get unpublished events: %v", err)
	}

	if len(events) == 0 {
		t.Fatal("expected at least one unpublished event")
	}

	// Find the transfer created event
	var transferEvent *domain.OutboxEvent
	for _, event := range events {
		if event.EventType == domain.EventTypeTransferCreated && event.AggregateID == transfer.ID {
			transferEvent = event
			break
		}
	}

	if transferEvent == nil {
		t.Fatal("transfer created event not found in outbox")
	}

	// Verify event details
	if transferEvent.AggregateType != domain.AggregateTypeTransfer {
		t.Errorf("expected aggregate type %s, got %s", domain.AggregateTypeTransfer, transferEvent.AggregateType)
	}

	if transferEvent.Published {
		t.Error("event should not be published yet")
	}

	if transferEvent.Payload == nil {
		t.Fatal("event payload is nil")
	}

	// Verify payload contains transfer details
	if transferEvent.Payload["transfer_id"] != transfer.ID {
		t.Errorf("payload transfer_id mismatch: expected %s, got %v", transfer.ID, transferEvent.Payload["transfer_id"])
	}

	if transferEvent.Payload["from_account_id"] != acc1.ID {
		t.Errorf("payload from_account_id mismatch")
	}

	if transferEvent.Payload["to_account_id"] != acc2.ID {
		t.Errorf("payload to_account_id mismatch")
	}
}

func TestEventPublisher(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	testDB := testutil.NewTestDB(t)
	defer testDB.Cleanup()

	pool := testDB.Pool
	accountRepo := postgres.NewAccountRepository(pool)
	transferRepo := postgres.NewTransferRepository(pool)
	entryRepo := postgres.NewEntryRepository(pool)
	outboxRepo := postgres.NewOutboxRepository(pool)
	txManager := postgres.NewTxManager(pool)
	idGen := postgres.NewULIDGenerator()
	retrier := postgres.NewRetrier()

	transferUC := usecase.NewTransferUseCase(txManager, accountRepo, transferRepo, entryRepo, outboxRepo, nil, idGen, nil).WithRetrier(retrier)

	// Create accounts and transfer to generate events
	acc1 := testDB.CreateTestAccountWithBalance(ctx, "acc1", "USD", decimal.NewFromInt(1000), false, true)
	acc2 := testDB.CreateTestAccount(ctx, "acc2", "USD", false, true)

	_, err := transferUC.CreateTransfer(ctx, usecase.CreateTransferInput{
		FromAccountID: acc1.ID,
		ToAccountID:   acc2.ID,
		Amount:        decimal.NewFromInt(100),
	})
	if err != nil {
		t.Fatalf("failed to create transfer: %v", err)
	}

	// Create event publisher
	mockPublisher := &MockPublisher{published: make([]*domain.OutboxEvent, 0)}
	publisher := eventpublisher.NewEventPublisher(eventpublisher.Config{
		OutboxRepo: outboxRepo,
		Publisher:  mockPublisher,
		BatchSize:  10,
	})

	// Process events once (not in background loop)
	publisherCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// Start publisher (it will process once and then wait)
	go publisher.Start(publisherCtx)

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)

	// Verify events were published
	published := mockPublisher.GetPublished()
	if len(published) == 0 {
		t.Fatal("no events were published")
	}

	// Verify events are marked as published in database
	unpublished, err := outboxRepo.GetUnpublished(ctx, 10)
	if err != nil {
		t.Fatalf("failed to get unpublished events: %v", err)
	}

	if len(unpublished) > 0 {
		t.Errorf("expected 0 unpublished events after publishing, got %d", len(unpublished))
	}
}

// MockPublisher for testing
type MockPublisher struct {
	mu        sync.Mutex
	published []*domain.OutboxEvent
}

func (m *MockPublisher) Publish(ctx context.Context, event *domain.OutboxEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.published = append(m.published, event)
	return nil
}

func (m *MockPublisher) GetPublished() []*domain.OutboxEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]*domain.OutboxEvent{}, m.published...)
}

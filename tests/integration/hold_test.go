package integration

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/iho/goledger/internal/adapter/repository/postgres"
	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/usecase"
	"github.com/iho/goledger/tests/testutil"
)

func TestHoldLifecycle(t *testing.T) {
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
	holdRepo := postgres.NewHoldRepository(pool)
	txManager := postgres.NewTxManager(pool)
	idGen := postgres.NewULIDGenerator()

	outboxRepo := postgres.NewNullOutboxRepository()
	holdUC := usecase.NewHoldUseCase(txManager, accountRepo, holdRepo, transferRepo, entryRepo, outboxRepo, nil, idGen, nil)

	t.Run("full hold lifecycle: create -> void", func(t *testing.T) {
		testDB.TruncateAll(ctx)

		// Create account with 100 balance
		acc := testDB.CreateTestAccountWithBalance(ctx, "acc1", "USD", decimal.NewFromInt(100), false, true)

		// 1. Create Hold of 50
		holdAmount := decimal.NewFromInt(50)
		hold, err := holdUC.HoldFunds(ctx, acc.ID, holdAmount)
		if err != nil {
			t.Fatalf("failed to create hold: %v", err)
		}

		// Verify account state
		updatedAcc, _ := accountRepo.GetByID(ctx, acc.ID)
		if !updatedAcc.Balance.Equal(decimal.NewFromInt(100)) {
			t.Errorf("expected balance 100, got %s", updatedAcc.Balance)
		}
		if !updatedAcc.EncumberedBalance.Equal(holdAmount) {
			t.Errorf("expected encumbered 50, got %s", updatedAcc.EncumberedBalance)
		}
		if !updatedAcc.AvailableBalance().Equal(decimal.NewFromInt(50)) {
			t.Errorf("expected available 50, got %s", updatedAcc.AvailableBalance())
		}

		// 2. Void Hold
		if err := holdUC.VoidHold(ctx, hold.ID); err != nil {
			t.Fatalf("failed to void hold: %v", err)
		}

		// Verify hold status
		updatedHold, _ := holdRepo.GetByID(ctx, hold.ID)
		if updatedHold.Status != domain.HoldStatusVoided {
			t.Errorf("expected status voided, got %s", updatedHold.Status)
		}

		// Verify account state (released funds)
		updatedAcc, _ = accountRepo.GetByID(ctx, acc.ID)
		if !updatedAcc.EncumberedBalance.IsZero() {
			t.Errorf("expected encumbered 0, got %s", updatedAcc.EncumberedBalance)
		}
		if !updatedAcc.AvailableBalance().Equal(decimal.NewFromInt(100)) {
			t.Errorf("expected available 100, got %s", updatedAcc.AvailableBalance())
		}
	})

	t.Run("full hold lifecycle: create -> capture", func(t *testing.T) {
		testDB.TruncateAll(ctx)

		// Create source with 100, dest with 0
		source := testDB.CreateTestAccountWithBalance(ctx, "source", "USD", decimal.NewFromInt(100), false, true)
		dest := testDB.CreateTestAccount(ctx, "dest", "USD", false, true)

		// 1. Create Hold of 50
		holdAmount := decimal.NewFromInt(50)
		hold, err := holdUC.HoldFunds(ctx, source.ID, holdAmount)
		if err != nil {
			t.Fatalf("failed to create hold: %v", err)
		}

		// 2. Capture Hold
		transfer, err := holdUC.CaptureHold(ctx, hold.ID, dest.ID)
		if err != nil {
			t.Fatalf("failed to capture hold: %v", err)
		}

		// Verify transfer
		if !transfer.Amount.Equal(holdAmount) {
			t.Errorf("expected transfer amount 50, got %s", transfer.Amount)
		}

		// Verify hold status
		updatedHold, _ := holdRepo.GetByID(ctx, hold.ID)
		if updatedHold.Status != domain.HoldStatusCaptured {
			t.Errorf("expected status captured, got %s", updatedHold.Status)
		}

		// Verify source account (balance reduced, encumbered released)
		sourceAcc, _ := accountRepo.GetByID(ctx, source.ID)
		if !sourceAcc.Balance.Equal(decimal.NewFromInt(50)) {
			t.Errorf("expected balance 50, got %s", sourceAcc.Balance)
		}
		if !sourceAcc.EncumberedBalance.IsZero() {
			t.Errorf("expected encumbered 0, got %s", sourceAcc.EncumberedBalance)
		}

		// Verify dest account
		destAcc, _ := accountRepo.GetByID(ctx, dest.ID)
		if !destAcc.Balance.Equal(decimal.NewFromInt(50)) {
			t.Errorf("expected balance 50, got %s", destAcc.Balance)
		}
	})

	t.Run("cannot hold more than available", func(t *testing.T) {
		testDB.TruncateAll(ctx)
		acc := testDB.CreateTestAccountWithBalance(ctx, "acc", "USD", decimal.NewFromInt(100), false, true)

		// Try to hold 150
		_, err := holdUC.HoldFunds(ctx, acc.ID, decimal.NewFromInt(150))
		if err != domain.ErrNegativeBalanceNotAllowed {
			t.Errorf("expected ErrNegativeBalanceNotAllowed, got %v", err)
		}
	})
}

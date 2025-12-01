package integration

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/iho/goledger/internal/adapter/repository/postgres"
	"github.com/iho/goledger/internal/usecase"
	"github.com/iho/goledger/tests/testutil"
)

func TestReverseTransfer(t *testing.T) {
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
	txManager := postgres.NewTxManager(pool)
	idGen := postgres.NewULIDGenerator()
	retrier := postgres.NewRetrier()

	transferUC := usecase.NewTransferUseCase(txManager, accountRepo, transferRepo, entryRepo, idGen).WithRetrier(retrier)

	// Create accounts with balance
	initialBalance := decimal.NewFromInt(1000)
	acc1 := testDB.CreateTestAccountWithBalance(ctx, "acc1", "USD", initialBalance, false, true)
	acc2 := testDB.CreateTestAccount(ctx, "acc2", "USD", false, true)

	// Create a transfer from account 1 to account 2
	transferAmount := decimal.NewFromInt(500)
	originalTransfer, err := transferUC.CreateTransfer(ctx, usecase.CreateTransferInput{
		FromAccountID: acc1.ID,
		ToAccountID:   acc2.ID,
		Amount:        transferAmount,
	})
	if err != nil {
		t.Fatalf("failed to create transfer: %v", err)
	}

	// Verify balances before reversal
	acc1AfterTransfer, err := accountRepo.GetByID(ctx, acc1.ID)
	if err != nil {
		t.Fatalf("failed to get account 1: %v", err)
	}
	if !acc1AfterTransfer.Balance.Equal(decimal.NewFromInt(500)) {
		t.Errorf("expected account 1 balance 500, got %s", acc1AfterTransfer.Balance)
	}

	acc2AfterTransfer, err := accountRepo.GetByID(ctx, acc2.ID)
	if err != nil {
		t.Fatalf("failed to get account 2: %v", err)
	}
	if !acc2AfterTransfer.Balance.Equal(decimal.NewFromInt(500)) {
		t.Errorf("expected account 2 balance 500, got %s", acc2AfterTransfer.Balance)
	}

	// Reverse the transfer
	reversalTransfer, err := transferUC.ReverseTransfer(ctx, usecase.ReverseTransferInput{
		TransferID: originalTransfer.ID,
		Metadata: map[string]any{
			"reason": "test reversal",
		},
	})
	if err != nil {
		t.Fatalf("failed to reverse transfer: %v", err)
	}
	if reversalTransfer == nil {
		t.Fatal("expected reversal transfer, got nil")
	}
	if reversalTransfer.ReversedTransferID == nil {
		t.Error("expected ReversedTransferID to be set")
	}
	if *reversalTransfer.ReversedTransferID != originalTransfer.ID {
		t.Errorf("expected ReversedTransferID %s, got %s", originalTransfer.ID, *reversalTransfer.ReversedTransferID)
	}

	// Verify the reversal has swapped from/to
	if reversalTransfer.FromAccountID != originalTransfer.ToAccountID {
		t.Errorf("expected FromAccountID %s, got %s", originalTransfer.ToAccountID, reversalTransfer.FromAccountID)
	}
	if reversalTransfer.ToAccountID != originalTransfer.FromAccountID {
		t.Errorf("expected ToAccountID %s, got %s", originalTransfer.FromAccountID, reversalTransfer.ToAccountID)
	}
	if !reversalTransfer.Amount.Equal(originalTransfer.Amount) {
		t.Errorf("expected amount %s, got %s", originalTransfer.Amount, reversalTransfer.Amount)
	}

	// Verify balances after reversal
	acc1AfterReversal, err := accountRepo.GetByID(ctx, acc1.ID)
	if err != nil {
		t.Fatalf("failed to get account 1 after reversal: %v", err)
	}
	if !acc1AfterReversal.Balance.Equal(initialBalance) {
		t.Errorf("expected account 1 balance %s, got %s", initialBalance, acc1AfterReversal.Balance)
	}

	acc2AfterReversal, err := accountRepo.GetByID(ctx, acc2.ID)
	if err != nil {
		t.Fatalf("failed to get account 2 after reversal: %v", err)
	}
	if !acc2AfterReversal.Balance.Equal(decimal.Zero) {
		t.Errorf("expected account 2 balance 0, got %s", acc2AfterReversal.Balance)
	}

	// Verify metadata
	if reversalTransfer.Metadata == nil {
		t.Error("expected metadata to be set")
	}
	if reversalTransfer.Metadata["reason"] != "test reversal" {
		t.Errorf("expected reason 'test reversal', got %v", reversalTransfer.Metadata["reason"])
	}
	if reversalTransfer.Metadata["reversal_of"] != originalTransfer.ID {
		t.Errorf("expected reversal_of %s, got %v", originalTransfer.ID, reversalTransfer.Metadata["reversal_of"])
	}
}

func TestReverseTransferTwice(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	testDB := testutil.NewTestDB(t)
	defer testDB.Cleanup()

	pool := testDB.Pool
	transferRepo := postgres.NewTransferRepository(pool)
	entryRepo := postgres.NewEntryRepository(pool)
	accountRepo := postgres.NewAccountRepository(pool)
	txManager := postgres.NewTxManager(pool)
	idGen := postgres.NewULIDGenerator()
	retrier := postgres.NewRetrier()

	transferUC := usecase.NewTransferUseCase(txManager, accountRepo, transferRepo, entryRepo, idGen).WithRetrier(retrier)

	// Create accounts and fund account 1
	acc1 := testDB.CreateTestAccountWithBalance(ctx, "acc1", "USD", decimal.NewFromInt(1000), false, true)
	acc2 := testDB.CreateTestAccount(ctx, "acc2", "USD", false, true)

	// Create a transfer
	originalTransfer, err := transferUC.CreateTransfer(ctx, usecase.CreateTransferInput{
		FromAccountID: acc1.ID,
		ToAccountID:   acc2.ID,
		Amount:        decimal.NewFromInt(500),
	})
	if err != nil {
		t.Fatalf("failed to create transfer: %v", err)
	}

	// Reverse the transfer once
	_, err = transferUC.ReverseTransfer(ctx, usecase.ReverseTransferInput{
		TransferID: originalTransfer.ID,
	})
	if err != nil {
		t.Fatalf("failed to reverse transfer: %v", err)
	}

	// Try to reverse the same transfer again - should fail
	_, err = transferUC.ReverseTransfer(ctx, usecase.ReverseTransferInput{
		TransferID: originalTransfer.ID,
	})
	if err == nil {
		t.Error("expected error when reversing transfer twice, got nil")
	}
}

func TestReverseTransferInsufficientBalance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	testDB := testutil.NewTestDB(t)
	defer testDB.Cleanup()

	pool := testDB.Pool
	transferRepo := postgres.NewTransferRepository(pool)
	entryRepo := postgres.NewEntryRepository(pool)
	accountRepo := postgres.NewAccountRepository(pool)
	txManager := postgres.NewTxManager(pool)
	idGen := postgres.NewULIDGenerator()
	retrier := postgres.NewRetrier()

	transferUC := usecase.NewTransferUseCase(txManager, accountRepo, transferRepo, entryRepo, idGen).WithRetrier(retrier)

	// Create accounts
	acc1 := testDB.CreateTestAccountWithBalance(ctx, "acc1", "USD", decimal.NewFromInt(1000), false, true)
	acc2 := testDB.CreateTestAccount(ctx, "acc2", "USD", false, true)

	// Transfer 500 from account 1 to account 2
	originalTransfer, err := transferUC.CreateTransfer(ctx, usecase.CreateTransferInput{
		FromAccountID: acc1.ID,
		ToAccountID:   acc2.ID,
		Amount:        decimal.NewFromInt(500),
	})
	if err != nil {
		t.Fatalf("failed to create transfer: %v", err)
	}

	// Account 2 now has 500, transfer it away
	_, err = transferUC.CreateTransfer(ctx, usecase.CreateTransferInput{
		FromAccountID: acc2.ID,
		ToAccountID:   acc1.ID,
		Amount:        decimal.NewFromInt(500),
	})
	if err != nil {
		t.Fatalf("failed to transfer funds away: %v", err)
	}

	// Now try to reverse the original transfer - account 2 has 0 balance, should fail
	_, err = transferUC.ReverseTransfer(ctx, usecase.ReverseTransferInput{
		TransferID: originalTransfer.ID,
	})
	if err == nil {
		t.Error("expected error for insufficient balance, got nil")
	}
}

func TestReverseNonExistentTransfer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	testDB := testutil.NewTestDB(t)
	defer testDB.Cleanup()

	pool := testDB.Pool
	transferRepo := postgres.NewTransferRepository(pool)
	entryRepo := postgres.NewEntryRepository(pool)
	accountRepo := postgres.NewAccountRepository(pool)
	txManager := postgres.NewTxManager(pool)
	idGen := postgres.NewULIDGenerator()
	retrier := postgres.NewRetrier()

	transferUC := usecase.NewTransferUseCase(txManager, accountRepo, transferRepo, entryRepo, idGen).WithRetrier(retrier)

	// Try to reverse a non-existent transfer
	_, err := transferUC.ReverseTransfer(ctx, usecase.ReverseTransferInput{
		TransferID: "non-existent-id",
	})
	if err == nil {
		t.Error("expected error for non-existent transfer, got nil")
	}
}

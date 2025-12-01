package integration

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/iho/goledger/internal/adapter/repository/postgres"
	"github.com/iho/goledger/internal/usecase"
	"github.com/iho/goledger/tests/testutil"
)

func TestConcurrentTransfers(t *testing.T) {
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

	outboxRepo := postgres.NewNullOutboxRepository()
	transferUC := usecase.NewTransferUseCase(txManager, accountRepo, transferRepo, entryRepo, outboxRepo, idGen)

	t.Run("100 concurrent transfers from same account no overdraft", func(t *testing.T) {
		testDB.TruncateAll(ctx)

		// Create source with balance that allows exactly 100 transfers of 10
		source := testDB.CreateTestAccountWithBalance(ctx, "source", "USD", decimal.NewFromInt(1000), false, true)
		dest := testDB.CreateTestAccount(ctx, "dest", "USD", false, true)

		numTransfers := 100
		transferAmount := decimal.NewFromInt(10)

		var (
			wg           sync.WaitGroup
			successCount atomic.Int32
			errorCount   atomic.Int32
		)

		wg.Add(numTransfers)

		for range numTransfers {
			go func() {
				defer wg.Done()

				_, err := transferUC.CreateTransfer(ctx, usecase.CreateTransferInput{
					FromAccountID: source.ID,
					ToAccountID:   dest.ID,
					Amount:        transferAmount,
				})
				if err != nil {
					errorCount.Add(1)
				} else {
					successCount.Add(1)
				}
			}()
		}

		wg.Wait()

		// All 100 should succeed (1000 / 10 = 100)
		if successCount.Load() != int32(numTransfers) {
			t.Errorf("expected %d successful transfers, got %d (errors: %d)", numTransfers, successCount.Load(), errorCount.Load())
		}

		// Verify final balances
		sourceAcc, _ := accountRepo.GetByID(ctx, source.ID)
		destAcc, _ := accountRepo.GetByID(ctx, dest.ID)

		if !sourceAcc.Balance.Equal(decimal.Zero) {
			t.Errorf("expected source balance 0, got %s", sourceAcc.Balance)
		}

		if !destAcc.Balance.Equal(decimal.NewFromInt(1000)) {
			t.Errorf("expected dest balance 1000, got %s", destAcc.Balance)
		}
	})

	t.Run("concurrent transfers reject overdraft", func(t *testing.T) {
		testDB.TruncateAll(ctx)

		// Create source with limited balance
		source := testDB.CreateTestAccountWithBalance(ctx, "source", "USD", decimal.NewFromInt(100), false, true)
		dest := testDB.CreateTestAccount(ctx, "dest", "USD", false, true)

		numTransfers := 20
		transferAmount := decimal.NewFromInt(10) // 20 * 10 = 200 > 100

		var (
			wg           sync.WaitGroup
			successCount atomic.Int32
			errorCount   atomic.Int32
		)

		wg.Add(numTransfers)

		for range numTransfers {
			go func() {
				defer wg.Done()

				_, err := transferUC.CreateTransfer(ctx, usecase.CreateTransferInput{
					FromAccountID: source.ID,
					ToAccountID:   dest.ID,
					Amount:        transferAmount,
				})
				if err != nil {
					errorCount.Add(1)
				} else {
					successCount.Add(1)
				}
			}()
		}

		wg.Wait()

		// Only 10 should succeed (100 / 10 = 10)
		if successCount.Load() != 10 {
			t.Errorf("expected 10 successful transfers, got %d", successCount.Load())
		}

		// Verify source is at 0
		sourceAcc, _ := accountRepo.GetByID(ctx, source.ID)
		if !sourceAcc.Balance.Equal(decimal.Zero) {
			t.Errorf("expected source balance 0, got %s", sourceAcc.Balance)
		}
	})

	t.Run("deadlock prevention with cross-account transfers", func(t *testing.T) {
		testDB.TruncateAll(ctx)

		// Create two accounts
		a := testDB.CreateTestAccountWithBalance(ctx, "a", "USD", decimal.NewFromInt(1000), true, true)
		b := testDB.CreateTestAccountWithBalance(ctx, "b", "USD", decimal.NewFromInt(1000), true, true)

		numTransfers := 50

		var (
			wg           sync.WaitGroup
			successCount atomic.Int32
		)

		// Half transfer A -> B, half transfer B -> A concurrently

		wg.Add(numTransfers * 2)

		for range numTransfers {
			go func() {
				defer wg.Done()

				_, err := transferUC.CreateTransfer(ctx, usecase.CreateTransferInput{
					FromAccountID: a.ID,
					ToAccountID:   b.ID,
					Amount:        decimal.NewFromInt(10),
				})
				if err == nil {
					successCount.Add(1)
				}
			}()
			go func() {
				defer wg.Done()

				_, err := transferUC.CreateTransfer(ctx, usecase.CreateTransferInput{
					FromAccountID: b.ID,
					ToAccountID:   a.ID,
					Amount:        decimal.NewFromInt(10),
				})
				if err == nil {
					successCount.Add(1)
				}
			}()
		}

		wg.Wait()

		// All transfers should succeed (no deadlock)
		if successCount.Load() != int32(numTransfers*2) {
			t.Errorf("expected %d successful transfers, got %d", numTransfers*2, successCount.Load())
		}

		// Balances should be unchanged (equal opposite transfers)
		aAcc, _ := accountRepo.GetByID(ctx, a.ID)
		bAcc, _ := accountRepo.GetByID(ctx, b.ID)

		if !aAcc.Balance.Equal(decimal.NewFromInt(1000)) {
			t.Errorf("expected a balance 1000, got %s", aAcc.Balance)
		}

		if !bAcc.Balance.Equal(decimal.NewFromInt(1000)) {
			t.Errorf("expected b balance 1000, got %s", bAcc.Balance)
		}
	})
}

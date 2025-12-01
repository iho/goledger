
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/shopspring/decimal"

	adaptershttp "github.com/iho/goledger/internal/adapter/http"
	"github.com/iho/goledger/internal/adapter/http/dto"
	"github.com/iho/goledger/internal/adapter/http/handler"
	"github.com/iho/goledger/internal/adapter/repository/postgres"
	redisrepo "github.com/iho/goledger/internal/adapter/repository/redis"
	infraredis "github.com/iho/goledger/internal/infrastructure/redis"
	"github.com/iho/goledger/internal/usecase"
	"github.com/iho/goledger/tests/testutil"
)

func TestTransfer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	testDB := testutil.NewTestDB(t)
	defer testDB.Cleanup()

	testDB.TruncateAll(ctx)

	pool := testDB.Pool
	accountRepo := postgres.NewAccountRepository(pool)
	transferRepo := postgres.NewTransferRepository(pool)
	entryRepo := postgres.NewEntryRepository(pool)
	txManager := postgres.NewTxManager(pool)
	idGen := postgres.NewULIDGenerator()

	accountUC := usecase.NewAccountUseCase(accountRepo, idGen)
	transferUC := usecase.NewTransferUseCase(txManager, accountRepo, transferRepo, entryRepo, idGen)
	entryUC := usecase.NewEntryUseCase(entryRepo)

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	redisClient, err := infraredis.NewClient(ctx, redisURL)
	if err != nil {
		t.Fatalf("failed to connect to redis: %v", err)
	}
	defer redisClient.Close()

	idempotencyStore := redisrepo.NewIdempotencyStore(redisClient)

	router := adaptershttp.NewRouter(adaptershttp.RouterConfig{
		AccountHandler:   handler.NewAccountHandler(accountUC),
		TransferHandler:  handler.NewTransferHandler(transferUC),
		EntryHandler:     handler.NewEntryHandler(entryUC),
		HealthHandler:    handler.NewHealthHandler(pool, redisClient),
		IdempotencyStore: idempotencyStore,
	})

	t.Run("create transfer between accounts", func(t *testing.T) {
		testDB.TruncateAll(ctx)

		// Create accounts - source allows negative, dest allows positive
		source := testDB.CreateTestAccountWithBalance(ctx, "source", "USD", decimal.NewFromInt(1000), true, true)
		dest := testDB.CreateTestAccount(ctx, "dest", "USD", false, true)

		req := dto.CreateTransferRequest{
			FromAccountID: source.ID,
			ToAccountID:   dest.ID,
			Amount:        "100.50",
		}
		body, _ := json.Marshal(req)

		r := httptest.NewRequest(http.MethodPost, "/api/v1/transfers", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		var resp dto.TransferResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		if err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if resp.Amount != "100.5" && resp.Amount != "100.50" {
			t.Errorf("expected amount 100.5, got %s", resp.Amount)
		}

		// Verify balances
		sourceAccount, _ := accountRepo.GetByID(ctx, source.ID)
		destAccount, _ := accountRepo.GetByID(ctx, dest.ID)

		expectedSource := decimal.NewFromFloat(899.50)
		if !sourceAccount.Balance.Equal(expectedSource) {
			t.Errorf("expected source balance %s, got %s", expectedSource, sourceAccount.Balance)
		}

		expectedDest := decimal.NewFromFloat(100.50)
		if !destAccount.Balance.Equal(expectedDest) {
			t.Errorf("expected dest balance %s, got %s", expectedDest, destAccount.Balance)
		}
	})

	t.Run("reject transfer to same account", func(t *testing.T) {
		testDB.TruncateAll(ctx)

		account := testDB.CreateTestAccountWithBalance(ctx, "self", "USD", decimal.NewFromInt(100), true, true)

		req := dto.CreateTransferRequest{
			FromAccountID: account.ID,
			ToAccountID:   account.ID,
			Amount:        "50",
		}
		body, _ := json.Marshal(req)

		r := httptest.NewRequest(http.MethodPost, "/api/v1/transfers", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("reject negative balance when not allowed", func(t *testing.T) {
		testDB.TruncateAll(ctx)

		// Source doesn't allow negative balance
		source := testDB.CreateTestAccountWithBalance(ctx, "source", "USD", decimal.NewFromInt(50), false, true)
		dest := testDB.CreateTestAccount(ctx, "dest", "USD", false, true)

		req := dto.CreateTransferRequest{
			FromAccountID: source.ID,
			ToAccountID:   dest.ID,
			Amount:        "100", // More than balance
		}
		body, _ := json.Marshal(req)

		r := httptest.NewRequest(http.MethodPost, "/api/v1/transfers", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})

	t.Run("reject currency mismatch", func(t *testing.T) {
		testDB.TruncateAll(ctx)

		source := testDB.CreateTestAccountWithBalance(ctx, "usd", "USD", decimal.NewFromInt(100), true, true)
		dest := testDB.CreateTestAccount(ctx, "eur", "EUR", false, true)

		req := dto.CreateTransferRequest{
			FromAccountID: source.ID,
			ToAccountID:   dest.ID,
			Amount:        "50",
		}
		body, _ := json.Marshal(req)

		r := httptest.NewRequest(http.MethodPost, "/api/v1/transfers", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
		}
	})

	t.Run("batch transfer atomic", func(t *testing.T) {
		testDB.TruncateAll(ctx)

		a := testDB.CreateTestAccountWithBalance(ctx, "a", "USD", decimal.NewFromInt(1000), true, true)
		b := testDB.CreateTestAccount(ctx, "b", "USD", true, true)
		c := testDB.CreateTestAccount(ctx, "c", "USD", true, true)

		req := dto.CreateBatchTransferRequest{
			Transfers: []dto.TransferItem{
				{FromAccountID: a.ID, ToAccountID: b.ID, Amount: "100"},
				{FromAccountID: a.ID, ToAccountID: c.ID, Amount: "200"},
			},
		}
		body, _ := json.Marshal(req)

		r := httptest.NewRequest(http.MethodPost, "/api/v1/transfers/batch", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		// Verify all balances
		aAcc, _ := accountRepo.GetByID(ctx, a.ID)
		bAcc, _ := accountRepo.GetByID(ctx, b.ID)
		cAcc, _ := accountRepo.GetByID(ctx, c.ID)

		if !aAcc.Balance.Equal(decimal.NewFromInt(700)) {
			t.Errorf("expected a balance 700, got %s", aAcc.Balance)
		}

		if !bAcc.Balance.Equal(decimal.NewFromInt(100)) {
			t.Errorf("expected b balance 100, got %s", bAcc.Balance)
		}

		if !cAcc.Balance.Equal(decimal.NewFromInt(200)) {
			t.Errorf("expected c balance 200, got %s", cAcc.Balance)
		}
	})

	t.Run("idempotency returns cached response", func(t *testing.T) {
		testDB.TruncateAll(ctx)

		source := testDB.CreateTestAccountWithBalance(ctx, "source", "USD", decimal.NewFromInt(1000), true, true)
		dest := testDB.CreateTestAccount(ctx, "dest", "USD", false, true)

		req := dto.CreateTransferRequest{
			FromAccountID: source.ID,
			ToAccountID:   dest.ID,
			Amount:        "100",
		}
		body, _ := json.Marshal(req)

		idempotencyKey := "test-key-" + testutil.GenerateID()

		// First request
		r1 := httptest.NewRequest(http.MethodPost, "/api/v1/transfers", bytes.NewReader(body))
		r1.Header.Set("Content-Type", "application/json")
		r1.Header.Set("Idempotency-Key", idempotencyKey)

		w1 := httptest.NewRecorder()

		router.ServeHTTP(w1, r1)

		if w1.Code != http.StatusCreated {
			t.Fatalf("first request failed: %d %s", w1.Code, w1.Body.String())
		}

		var resp1 dto.TransferResponse
		json.Unmarshal(w1.Body.Bytes(), &resp1)

		// Second request with same key
		body2, _ := json.Marshal(req)
		r2 := httptest.NewRequest(http.MethodPost, "/api/v1/transfers", bytes.NewReader(body2))
		r2.Header.Set("Content-Type", "application/json")
		r2.Header.Set("Idempotency-Key", idempotencyKey)

		w2 := httptest.NewRecorder()

		router.ServeHTTP(w2, r2)

		if w2.Code != http.StatusOK && w2.Code != http.StatusCreated {
			t.Errorf("second request failed: %d %s", w2.Code, w2.Body.String())
		}

		var resp2 dto.TransferResponse
		json.Unmarshal(w2.Body.Bytes(), &resp2)

		// Should return same transfer ID
		if resp1.ID != resp2.ID {
			t.Errorf("expected same transfer ID, got %s vs %s", resp1.ID, resp2.ID)
		}

		// Balance should only be debited once
		sourceAcc, _ := accountRepo.GetByID(ctx, source.ID)
		if !sourceAcc.Balance.Equal(decimal.NewFromInt(900)) {
			t.Errorf("expected balance 900 (debited once), got %s", sourceAcc.Balance)
		}
	})
}

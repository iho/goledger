
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

func TestEdgeCases(t *testing.T) {
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

	t.Run("transfer exact balance leaving zero", func(t *testing.T) {
		testDB.TruncateAll(ctx)

		// Create account with exactly 100, no negative allowed
		source := testDB.CreateTestAccountWithBalance(ctx, "exact", "USD", decimal.NewFromInt(100), false, true)
		dest := testDB.CreateTestAccount(ctx, "dest", "USD", false, true)

		req := dto.CreateTransferRequest{
			FromAccountID: source.ID,
			ToAccountID:   dest.ID,
			Amount:        "100", // Exact balance
		}
		body, _ := json.Marshal(req)

		r := httptest.NewRequest(http.MethodPost, "/api/v1/transfers", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		// Verify source is exactly zero
		sourceAcc, _ := accountRepo.GetByID(ctx, source.ID)
		if !sourceAcc.Balance.IsZero() {
			t.Errorf("expected source balance 0, got %s", sourceAcc.Balance)
		}

		// Verify dest has 100
		destAcc, _ := accountRepo.GetByID(ctx, dest.ID)
		if !destAcc.Balance.Equal(decimal.NewFromInt(100)) {
			t.Errorf("expected dest balance 100, got %s", destAcc.Balance)
		}
	})

	t.Run("very large decimal amounts", func(t *testing.T) {
		testDB.TruncateAll(ctx)

		// Test with large precision decimals
		largeAmount := decimal.RequireFromString("999999999999.999999999999")
		source := testDB.CreateTestAccountWithBalance(ctx, "large", "USD", largeAmount, true, true)
		dest := testDB.CreateTestAccount(ctx, "dest", "USD", true, true)

		transferAmount := "123456789.123456789"
		req := dto.CreateTransferRequest{
			FromAccountID: source.ID,
			ToAccountID:   dest.ID,
			Amount:        transferAmount,
		}
		body, _ := json.Marshal(req)

		r := httptest.NewRequest(http.MethodPost, "/api/v1/transfers", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		// Verify balances are correct
		sourceAcc, _ := accountRepo.GetByID(ctx, source.ID)

		expectedSource := largeAmount.Sub(decimal.RequireFromString(transferAmount))
		if !sourceAcc.Balance.Equal(expectedSource) {
			t.Errorf("expected source balance %s, got %s", expectedSource, sourceAcc.Balance)
		}
	})

	t.Run("unicode in account names", func(t *testing.T) {
		testDB.TruncateAll(ctx)

		unicodeNames := []string{
			"日本語アカウント",
			"Контр агент",
			"Émile's Account 💰",
			"账户名称",
			"حساب عربي",
		}

		for _, name := range unicodeNames {
			req := dto.CreateAccountRequest{
				Name:                 name,
				Currency:             "USD",
				AllowNegativeBalance: true,
				AllowPositiveBalance: true,
			}
			body, _ := json.Marshal(req)

			r := httptest.NewRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(body))
			r.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			router.ServeHTTP(w, r)

			if w.Code != http.StatusCreated {
				t.Errorf("failed to create account with name %q: %d %s", name, w.Code, w.Body.String())
				continue
			}

			var resp dto.AccountResponse
			json.Unmarshal(w.Body.Bytes(), &resp)

			if resp.Name != name {
				t.Errorf("expected name %q, got %q", name, resp.Name)
			}
		}
	})

	t.Run("transfer with metadata", func(t *testing.T) {
		testDB.TruncateAll(ctx)

		source := testDB.CreateTestAccountWithBalance(ctx, "source", "USD", decimal.NewFromInt(1000), true, true)
		dest := testDB.CreateTestAccount(ctx, "dest", "USD", true, true)

		// Transfer with complex metadata
		reqBody := map[string]any{
			"from_account_id": source.ID,
			"to_account_id":   dest.ID,
			"amount":          "100",
			"metadata": map[string]any{
				"reference":    "INV-2024-001",
				"description":  "Payment for services",
				"tags":         []string{"invoice", "payment"},
				"nested":       map[string]any{"key": "value"},
				"unicode_note": "支付备注 📝",
			},
		}
		body, _ := json.Marshal(reqBody)

		r := httptest.NewRequest(http.MethodPost, "/api/v1/transfers", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		var resp dto.TransferResponse
		json.Unmarshal(w.Body.Bytes(), &resp)

		// Verify metadata was stored
		if resp.Metadata == nil {
			t.Error("expected metadata, got nil")
		}

		if resp.Metadata["reference"] != "INV-2024-001" {
			t.Errorf("expected reference INV-2024-001, got %v", resp.Metadata["reference"])
		}
	})

	t.Run("transfer with nil/empty metadata", func(t *testing.T) {
		testDB.TruncateAll(ctx)

		source := testDB.CreateTestAccountWithBalance(ctx, "source", "USD", decimal.NewFromInt(1000), true, true)
		dest := testDB.CreateTestAccount(ctx, "dest", "USD", true, true)

		// Transfer without metadata
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

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		// Should succeed even without metadata
		var resp dto.TransferResponse
		json.Unmarshal(w.Body.Bytes(), &resp)

		if resp.ID == "" {
			t.Error("expected transfer ID")
		}
	})

	t.Run("reject zero amount transfer", func(t *testing.T) {
		testDB.TruncateAll(ctx)

		source := testDB.CreateTestAccountWithBalance(ctx, "source", "USD", decimal.NewFromInt(100), true, true)
		dest := testDB.CreateTestAccount(ctx, "dest", "USD", true, true)

		req := dto.CreateTransferRequest{
			FromAccountID: source.ID,
			ToAccountID:   dest.ID,
			Amount:        "0",
		}
		body, _ := json.Marshal(req)

		r := httptest.NewRequest(http.MethodPost, "/api/v1/transfers", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d for zero amount, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("reject negative amount transfer", func(t *testing.T) {
		testDB.TruncateAll(ctx)

		source := testDB.CreateTestAccountWithBalance(ctx, "source", "USD", decimal.NewFromInt(100), true, true)
		dest := testDB.CreateTestAccount(ctx, "dest", "USD", true, true)

		req := dto.CreateTransferRequest{
			FromAccountID: source.ID,
			ToAccountID:   dest.ID,
			Amount:        "-50",
		}
		body, _ := json.Marshal(req)

		r := httptest.NewRequest(http.MethodPost, "/api/v1/transfers", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d for negative amount, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("verify entries created for transfer", func(t *testing.T) {
		testDB.TruncateAll(ctx)

		source := testDB.CreateTestAccountWithBalance(ctx, "source", "USD", decimal.NewFromInt(500), true, true)
		dest := testDB.CreateTestAccount(ctx, "dest", "USD", true, true)

		req := dto.CreateTransferRequest{
			FromAccountID: source.ID,
			ToAccountID:   dest.ID,
			Amount:        "200",
		}
		body, _ := json.Marshal(req)

		r := httptest.NewRequest(http.MethodPost, "/api/v1/transfers", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		var transferResp dto.TransferResponse
		json.Unmarshal(w.Body.Bytes(), &transferResp)

		// Get entries for the transfer
		r2 := httptest.NewRequest(http.MethodGet, "/api/v1/transfers/"+transferResp.ID+"/entries", http.NoBody)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, r2)

		if w2.Code != http.StatusOK {
			t.Fatalf("failed to get entries: %d %s", w2.Code, w2.Body.String())
		}

		var entries []dto.EntryResponse
		json.Unmarshal(w2.Body.Bytes(), &entries)

		if len(entries) != 2 {
			t.Fatalf("expected 2 entries (debit + credit), got %d", len(entries))
		}

		// Verify one is debit (-200) and one is credit (+200)
		var hasDebit, hasCredit bool
		for _, e := range entries {
			amount, _ := decimal.NewFromString(e.Amount)
			if amount.Equal(decimal.NewFromInt(-200)) {
				hasDebit = true

				if e.AccountID != source.ID {
					t.Errorf("debit entry should be for source account")
				}
			}

			if amount.Equal(decimal.NewFromInt(200)) {
				hasCredit = true

				if e.AccountID != dest.ID {
					t.Errorf("credit entry should be for dest account")
				}
			}
		}

		if !hasDebit {
			t.Error("missing debit entry")
		}

		if !hasCredit {
			t.Error("missing credit entry")
		}
	})

	t.Run("verify account version increments", func(t *testing.T) {
		testDB.TruncateAll(ctx)

		source := testDB.CreateTestAccountWithBalance(ctx, "source", "USD", decimal.NewFromInt(1000), true, true)
		dest := testDB.CreateTestAccount(ctx, "dest", "USD", true, true)

		// Initial version should be 0
		sourceAcc, _ := accountRepo.GetByID(ctx, source.ID)
		if sourceAcc.Version != 0 {
			t.Errorf("expected initial version 0, got %d", sourceAcc.Version)
		}

		// Make 3 transfers
		for range 3 {
			req := dto.CreateTransferRequest{
				FromAccountID: source.ID,
				ToAccountID:   dest.ID,
				Amount:        "100",
			}
			body, _ := json.Marshal(req)
			r := httptest.NewRequest(http.MethodPost, "/api/v1/transfers", bytes.NewReader(body))
			r.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
		}

		// Version should be 3
		sourceAcc, _ = accountRepo.GetByID(ctx, source.ID)
		if sourceAcc.Version != 3 {
			t.Errorf("expected version 3 after 3 transfers, got %d", sourceAcc.Version)
		}

		destAcc, _ := accountRepo.GetByID(ctx, dest.ID)
		if destAcc.Version != 3 {
			t.Errorf("expected dest version 3, got %d", destAcc.Version)
		}
	})
}

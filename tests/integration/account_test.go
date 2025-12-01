package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	adaptershttp "github.com/iho/goledger/internal/adapter/http"
	"github.com/iho/goledger/internal/adapter/http/dto"
	"github.com/iho/goledger/internal/adapter/http/handler"
	"github.com/iho/goledger/internal/adapter/repository/postgres"
	redisrepo "github.com/iho/goledger/internal/adapter/repository/redis"
	infraredis "github.com/iho/goledger/internal/infrastructure/redis"
	"github.com/iho/goledger/internal/usecase"
	"github.com/iho/goledger/tests/testutil"
)

func TestAccountCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	testDB := testutil.NewTestDB(t)
	defer testDB.Cleanup()
	testDB.TruncateAll(ctx)

	// Setup
	pool := testDB.Pool
	accountRepo := postgres.NewAccountRepository(pool)
	idGen := postgres.NewULIDGenerator()
	accountUC := usecase.NewAccountUseCase(accountRepo, idGen)
	accountHandler := handler.NewAccountHandler(accountUC)

	// Create router with just account handler
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
	transferUC := usecase.NewTransferUseCase(
		postgres.NewTxManager(pool),
		accountRepo,
		postgres.NewTransferRepository(pool),
		postgres.NewEntryRepository(pool),
		idGen,
	)
	entryUC := usecase.NewEntryUseCase(postgres.NewEntryRepository(pool))

	router := adaptershttp.NewRouter(adaptershttp.RouterConfig{
		AccountHandler:   accountHandler,
		TransferHandler:  handler.NewTransferHandler(transferUC),
		EntryHandler:     handler.NewEntryHandler(entryUC),
		HealthHandler:    handler.NewHealthHandler(pool, redisClient),
		IdempotencyStore: idempotencyStore,
	})

	t.Run("create account with valid data", func(t *testing.T) {
		req := dto.CreateAccountRequest{
			Name:                 "test-account",
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
			t.Errorf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		var resp dto.AccountResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if resp.Name != req.Name {
			t.Errorf("expected name %q, got %q", req.Name, resp.Name)
		}
		if resp.Currency != req.Currency {
			t.Errorf("expected currency %q, got %q", req.Currency, resp.Currency)
		}
		if resp.Balance != "0" {
			t.Errorf("expected balance 0, got %s", resp.Balance)
		}
	})

	t.Run("get account by ID", func(t *testing.T) {
		// First create an account
		account := testDB.CreateTestAccount(ctx, "get-test", "EUR", false, true)

		r := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/"+account.ID, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var resp dto.AccountResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if resp.ID != account.ID {
			t.Errorf("expected ID %q, got %q", account.ID, resp.ID)
		}
	})

	t.Run("get non-existent account returns 404", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/non-existent-id", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("list accounts", func(t *testing.T) {
		testDB.TruncateAll(ctx)
		testDB.CreateTestAccount(ctx, "list-1", "USD", true, true)
		testDB.CreateTestAccount(ctx, "list-2", "USD", true, true)

		r := httptest.NewRequest(http.MethodGet, "/api/v1/accounts?limit=10&offset=0", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var resp dto.ListAccountsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if len(resp.Accounts) != 2 {
			t.Errorf("expected 2 accounts, got %d", len(resp.Accounts))
		}
	})
}

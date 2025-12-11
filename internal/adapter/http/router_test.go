package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"github.com/iho/goledger/internal/adapter/http/handler"
	apimiddleware "github.com/iho/goledger/internal/adapter/http/middleware"
	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/usecase"
)

func TestNewRouter_HealthEndpointAvailable(t *testing.T) {
	router := NewRouter(newRouterConfig())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected /health to return 200, got %d", rec.Code)
	}
}

func TestNewRouter_RateLimiterBlocksExcessRequests(t *testing.T) {
	rl := apimiddleware.NewRateLimiter(1, 1)
	router := NewRouter(newRouterConfig(func(cfg *RouterConfig) {
		cfg.RateLimiter = rl
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/health", nil)
	req1.RemoteAddr = "1.2.3.4:1234"
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected first request to succeed, got %d", rec1.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/health", nil)
	req2.RemoteAddr = "1.2.3.4:1234"
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request to be throttled, got %d", rec2.Code)
	}
}

func TestNewRouter_IdempotencyMiddlewareInvokesStore(t *testing.T) {
	store := &stubIdempotencyStore{}
	router := NewRouter(newRouterConfig(func(cfg *RouterConfig) {
		cfg.IdempotencyStore = store
	}))

	body := `{"name":"Main","currency":"USD"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(apimiddleware.IdempotencyKeyHeader, "key-123")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if !store.checkCalled {
		t.Fatalf("expected idempotency store to be used")
	}
}

func TestNewRouter_RegistersKeyRoutes(t *testing.T) {
	router := NewRouter(newRouterConfig())

	chiRoutes, ok := router.(chi.Router)
	if !ok {
		t.Fatal("router does not implement chi.Routes")
	}

	seen := map[string]bool{}
	if err := chi.Walk(chiRoutes, func(method string, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		seen[method+" "+route] = true
		return nil
	}); err != nil {
		t.Fatalf("walk failed: %v", err)
	}

	expected := []string{
		"GET /health",
		"GET /ready",
		"POST /api/v1/accounts/",
		"GET /api/v1/accounts/",
		"GET /api/v1/accounts/{id}",
		"POST /api/v1/transfers/",
		"POST /api/v1/holds/",
	}

	for _, route := range expected {
		if !seen[route] {
			t.Fatalf("expected route %s to be registered", route)
		}
	}
}

func newRouterConfig(opts ...func(*RouterConfig)) RouterConfig {
	accountHandler := handler.NewAccountHandler(&stubAccountService{})
	transferHandler := handler.NewTransferHandler(&stubTransferService{})

	entryRepo := &stubEntryRepository{}
	entryUC := usecase.NewEntryUseCase(entryRepo)
	entryHandler := handler.NewEntryHandler(entryUC)

	ledgerHandler := handler.NewLedgerHandler(usecase.NewLedgerUseCase(&stubLedgerRepository{}))
	holdHandler := handler.NewHoldHandler(nil)

	cfg := RouterConfig{
		HealthHandler:   &handler.HealthHandler{},
		AccountHandler:  accountHandler,
		TransferHandler: transferHandler,
		EntryHandler:    entryHandler,
		LedgerHandler:   ledgerHandler,
		HoldHandler:     holdHandler,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	return cfg
}

type stubAccountService struct{}

func (stubAccountService) CreateAccount(ctx context.Context, input usecase.CreateAccountInput) (*domain.Account, error) {
	return &domain.Account{ID: "acc"}, nil
}

func (stubAccountService) GetAccount(ctx context.Context, id string) (*domain.Account, error) {
	return &domain.Account{ID: id}, nil
}

func (stubAccountService) ListAccounts(ctx context.Context, input usecase.ListAccountsInput) ([]*domain.Account, error) {
	return []*domain.Account{}, nil
}

type stubTransferService struct{}

func (stubTransferService) CreateTransfer(ctx context.Context, input usecase.CreateTransferInput) (*domain.Transfer, error) {
	return &domain.Transfer{ID: "transfer"}, nil
}

func (stubTransferService) CreateBatchTransfer(ctx context.Context, input usecase.CreateBatchTransferInput) ([]*domain.Transfer, error) {
	return []*domain.Transfer{}, nil
}

func (stubTransferService) GetTransfer(ctx context.Context, id string) (*domain.Transfer, error) {
	return &domain.Transfer{ID: id}, nil
}

func (stubTransferService) ListTransfersByAccount(ctx context.Context, input usecase.ListTransfersByAccountInput) ([]*domain.Transfer, error) {
	return []*domain.Transfer{}, nil
}

func (stubTransferService) ReverseTransfer(ctx context.Context, input usecase.ReverseTransferInput) (*domain.Transfer, error) {
	return &domain.Transfer{ID: input.TransferID}, nil
}

type stubEntryRepository struct{}

func (stubEntryRepository) Create(ctx context.Context, tx usecase.Transaction, entry *domain.Entry) error {
	return nil
}

func (stubEntryRepository) GetByTransfer(ctx context.Context, transferID string) ([]*domain.Entry, error) {
	return []*domain.Entry{}, nil
}

func (stubEntryRepository) GetByAccount(ctx context.Context, accountID string, limit, offset int) ([]*domain.Entry, error) {
	return []*domain.Entry{}, nil
}

func (stubEntryRepository) GetBalanceAtTime(ctx context.Context, accountID string, at time.Time) (decimal.Decimal, error) {
	return decimal.Zero, nil
}

type stubLedgerRepository struct{}

func (stubLedgerRepository) CheckConsistency(ctx context.Context) (decimal.Decimal, decimal.Decimal, error) {
	return decimal.Zero, decimal.Zero, nil
}

type stubIdempotencyStore struct {
	checkCalled bool
}

func (s *stubIdempotencyStore) CheckAndSet(ctx context.Context, key string, response []byte, ttl time.Duration) (bool, []byte, error) {
	s.checkCalled = true
	return false, nil, nil
}

func (s *stubIdempotencyStore) Update(ctx context.Context, key string, response []byte, ttl time.Duration) error {
	return nil
}

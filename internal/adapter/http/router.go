package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"log/slog"

	"github.com/iho/goledger/internal/adapter/http/handler"
	"github.com/iho/goledger/internal/adapter/http/middleware"
	"github.com/iho/goledger/internal/usecase"
)

// RouterConfig holds dependencies for the router.
type RouterConfig struct {
	AccountHandler   *handler.AccountHandler
	TransferHandler  *handler.TransferHandler
	EntryHandler     *handler.EntryHandler
	HealthHandler    *handler.HealthHandler
	LedgerHandler    *handler.LedgerHandler
	HoldHandler      *handler.HoldHandler
	IdempotencyStore usecase.IdempotencyStore
	Logger           *slog.Logger
}

// NewRouter creates a new HTTP router.
func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.Metrics)
	if cfg.Logger != nil {
		r.Use(middleware.NewLoggingMiddleware(cfg.Logger).Wrap)
	} else {
		r.Use(chimiddleware.Logger)
	}
	r.Use(chimiddleware.Recoverer)

	// Health & metrics endpoints
	r.Get("/health", cfg.HealthHandler.Liveness)
	r.Get("/ready", cfg.HealthHandler.Readiness)
	r.Handle("/metrics", promhttp.Handler())

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Idempotency middleware for mutating requests
		if cfg.IdempotencyStore != nil {
			idempotencyMiddleware := middleware.NewIdempotencyMiddleware(cfg.IdempotencyStore)
			r.Use(idempotencyMiddleware.Wrap)
		}

		// Ledger endpoints
		r.Get("/ledger/consistency", cfg.LedgerHandler.CheckConsistency)

		// Accounts
		r.Route("/accounts", func(r chi.Router) {
			r.Post("/", cfg.AccountHandler.Create)
			r.Get("/", cfg.AccountHandler.List)
			r.Get("/{id}", cfg.AccountHandler.Get)
			r.Get("/{id}/entries", cfg.EntryHandler.ListByAccount)
			r.Get("/{id}/transfers", cfg.TransferHandler.ListByAccount)
			r.Get("/{id}/balance/history", cfg.EntryHandler.GetHistoricalBalance)
		})

		// Transfers
		r.Route("/transfers", func(r chi.Router) {
			r.Post("/", cfg.TransferHandler.Create)
			r.Post("/batch", cfg.TransferHandler.CreateBatch)
			r.Get("/{id}", cfg.TransferHandler.Get)
			r.Get("/{id}/entries", cfg.EntryHandler.ListByTransfer)
			r.Post("/{id}/reverse", cfg.TransferHandler.Reverse)
		})

		// Holds
		r.Route("/holds", func(r chi.Router) {
			r.Post("/", cfg.HoldHandler.Create)
			r.Post("/{id}/void", cfg.HoldHandler.Void)
			r.Post("/{id}/capture", cfg.HoldHandler.Capture)
		})
	})

	return r
}

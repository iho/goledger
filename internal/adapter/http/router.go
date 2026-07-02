package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"log/slog"

	"github.com/iho/goledger/internal/adapter/http/handler"
	"github.com/iho/goledger/internal/adapter/http/middleware"
	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/infrastructure/auth"
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
	AuthHandler      *handler.AuthHandler
	AuditHandler     *handler.AuditHandler
	IdempotencyStore usecase.IdempotencyStore
	RateLimiter      *middleware.RateLimiter
	Logger           *slog.Logger
	// JWTManager verifies bearer tokens. Required for auth enforcement.
	JWTManager *auth.JWTManager
	// AuthEnabled turns on authentication/RBAC enforcement for the API. When
	// false (or JWTManager is nil), routes behave as before: open, no user
	// in context. Mirrors the AUTH_ENABLED config flag.
	AuthEnabled bool
}

// authRequired reports whether auth middleware should be applied.
func (cfg RouterConfig) authRequired() bool {
	return cfg.AuthEnabled && cfg.JWTManager != nil
}

// requireAuth returns AuthMiddleware if auth is enabled, otherwise a no-op
// pass-through so routes work unchanged when auth is disabled.
func requireAuth(cfg RouterConfig) func(http.Handler) http.Handler {
	if !cfg.authRequired() {
		return func(next http.Handler) http.Handler { return next }
	}
	return middleware.AuthMiddleware(cfg.JWTManager)
}

// requireRole returns RequireRole(role) if auth is enabled, otherwise a
// no-op pass-through.
func requireRole(cfg RouterConfig, role domain.Role) func(http.Handler) http.Handler {
	if !cfg.authRequired() {
		return func(next http.Handler) http.Handler { return next }
	}
	return middleware.RequireRole(role)
}

// NewRouter creates a new HTTP router.
func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.RequestMeta)
	if cfg.RateLimiter != nil {
		r.Use(cfg.RateLimiter.Limit)
	}
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

		// Login is public; every other /api/v1 route requires authentication
		// when AuthEnabled is set (see requireAuth/requireRole above).
		if cfg.AuthHandler != nil {
			r.Post("/auth/login", cfg.AuthHandler.Login)
		}

		r.Group(func(r chi.Router) {
			r.Use(requireAuth(cfg))

			if cfg.AuthHandler != nil {
				r.Get("/auth/me", cfg.AuthHandler.GetCurrentUser)
			}

			// Ledger endpoints - any authenticated role may view.
			r.Get("/ledger/consistency", cfg.LedgerHandler.CheckConsistency)

			// Accounts - creation is admin-only, viewing is open to all roles.
			r.Route("/accounts", func(r chi.Router) {
				r.With(requireRole(cfg, domain.RoleAdmin)).Post("/", cfg.AccountHandler.Create)
				r.Get("/", cfg.AccountHandler.List)
				r.Get("/{id}", cfg.AccountHandler.Get)
				r.Get("/{id}/entries", cfg.EntryHandler.ListByAccount)
				r.Get("/{id}/transfers", cfg.TransferHandler.ListByAccount)
				r.Get("/{id}/balance/history", cfg.EntryHandler.GetHistoricalBalance)
			})

			// Transfers - mutations require operator (or admin), viewing is open.
			r.Route("/transfers", func(r chi.Router) {
				r.With(requireRole(cfg, domain.RoleOperator)).Post("/", cfg.TransferHandler.Create)
				r.With(requireRole(cfg, domain.RoleOperator)).Post("/batch", cfg.TransferHandler.CreateBatch)
				r.Get("/{id}", cfg.TransferHandler.Get)
				r.Get("/{id}/entries", cfg.EntryHandler.ListByTransfer)
				r.With(requireRole(cfg, domain.RoleOperator)).Post("/{id}/reverse", cfg.TransferHandler.Reverse)
			})

			// Holds - mutations require operator (or admin).
			r.Route("/holds", func(r chi.Router) {
				r.With(requireRole(cfg, domain.RoleOperator)).Post("/", cfg.HoldHandler.Create)
				r.With(requireRole(cfg, domain.RoleOperator)).Post("/{id}/void", cfg.HoldHandler.Void)
				r.With(requireRole(cfg, domain.RoleOperator)).Post("/{id}/capture", cfg.HoldHandler.Capture)
			})

			// Audit - admin-only read access for examiners.
			if cfg.AuditHandler != nil {
				r.Route("/audit", func(r chi.Router) {
					r.Use(requireRole(cfg, domain.RoleAdmin))
					r.Get("/", cfg.AuditHandler.List)
					r.Get("/export", cfg.AuditHandler.Export)
					r.Get("/resource/{type}/{id}", cfg.AuditHandler.GetByResource)
					r.Get("/user/{userId}", cfg.AuditHandler.GetByUser)
				})
			}
		})
	})

	return r
}

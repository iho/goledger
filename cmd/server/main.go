package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	httpAdapter "github.com/iho/goledger/internal/adapter/http"
	"github.com/iho/goledger/internal/adapter/http/handler"
	postgresRepo "github.com/iho/goledger/internal/adapter/repository/postgres"
	redisRepo "github.com/iho/goledger/internal/adapter/repository/redis"
	"github.com/iho/goledger/internal/infrastructure/config"
	"github.com/iho/goledger/internal/infrastructure/postgres"
	"github.com/iho/goledger/internal/infrastructure/redis"
	"github.com/iho/goledger/internal/usecase"
)

func main() {
	// Setup logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load configuration")
	}

	// Setup log level
	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(level)

	ctx := context.Background()

	// Connect to PostgreSQL
	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL, cfg.DatabaseMaxConns, cfg.DatabaseMinConns)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to postgres")
	}
	defer pool.Close()

	log.Info().Msg("connected to postgres")

	// Run migrations
	if err := postgres.RunMigrations(cfg.DatabaseURL, "internal/infrastructure/postgres/migrations"); err != nil {
		log.Fatal().Err(err).Msg("failed to run migrations")
	}

	// Connect to Redis
	redisClient, err := redis.NewClient(ctx, cfg.RedisURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to redis")
	}
	defer redisClient.Close()

	log.Info().Msg("connected to redis")

	// Initialize repositories
	txManager := postgresRepo.NewTxManager(pool)
	accountRepo := postgresRepo.NewAccountRepository(pool)
	transferRepo := postgresRepo.NewTransferRepository(pool)
	entryRepo := postgresRepo.NewEntryRepository(pool)
	ledgerRepo := postgresRepo.NewLedgerRepository(pool)
	idempotencyStore := redisRepo.NewIdempotencyStore(redisClient)
	idGen := postgresRepo.NewULIDGenerator()

	// Initialize use cases with retry support
	retrier := postgresRepo.NewRetrier()
	accountUC := usecase.NewAccountUseCase(accountRepo, idGen)
	transferUC := usecase.NewTransferUseCase(txManager, accountRepo, transferRepo, entryRepo, idGen).
		WithRetrier(retrier)
	entryUC := usecase.NewEntryUseCase(entryRepo)
	ledgerUC := usecase.NewLedgerUseCase(ledgerRepo)

	// Initialize handlers
	accountHandler := handler.NewAccountHandler(accountUC)
	transferHandler := handler.NewTransferHandler(transferUC)
	entryHandler := handler.NewEntryHandler(entryUC)
	ledgerHandler := handler.NewLedgerHandler(ledgerUC)
	healthHandler := handler.NewHealthHandler(pool, redisClient)

	// Create router
	router := httpAdapter.NewRouter(httpAdapter.RouterConfig{
		AccountHandler:   accountHandler,
		TransferHandler:  transferHandler,
		EntryHandler:     entryHandler,
		HealthHandler:    healthHandler,
		LedgerHandler:    ledgerHandler,
		IdempotencyStore: idempotencyStore,
	})

	// Create server with timeouts
	server := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      router,
		ReadTimeout:  cfg.HTTPReadTimeout,
		WriteTimeout: cfg.HTTPWriteTimeout,
		IdleTimeout:  cfg.HTTPIdleTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Info().Str("port", cfg.HTTPPort).Msg("starting server")
		err := server.ListenAndServe()

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("server failed")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down server...")

	// Graceful shutdown with configured timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTPShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("server forced to shutdown")
	}

	log.Info().Msg("server stopped")
}

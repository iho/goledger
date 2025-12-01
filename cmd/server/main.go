package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

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
	idempotencyStore := redisRepo.NewIdempotencyStore(redisClient)
	idGen := postgresRepo.NewULIDGenerator()

	// Initialize use cases
	accountUC := usecase.NewAccountUseCase(accountRepo, idGen)
	transferUC := usecase.NewTransferUseCase(txManager, accountRepo, transferRepo, entryRepo, idGen)
	entryUC := usecase.NewEntryUseCase(entryRepo)

	// Initialize handlers
	accountHandler := handler.NewAccountHandler(accountUC)
	transferHandler := handler.NewTransferHandler(transferUC)
	entryHandler := handler.NewEntryHandler(entryUC)
	healthHandler := handler.NewHealthHandler(pool, redisClient)

	// Create router
	router := httpAdapter.NewRouter(httpAdapter.RouterConfig{
		AccountHandler:   accountHandler,
		TransferHandler:  transferHandler,
		EntryHandler:     entryHandler,
		HealthHandler:    healthHandler,
		IdempotencyStore: idempotencyStore,
	})

	// Create server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler:      router,
		ReadTimeout:  cfg.HTTPReadTimeout,
		WriteTimeout: cfg.HTTPWriteTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Info().Str("port", cfg.HTTPPort).Msg("starting server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server failed")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("server forced to shutdown")
	}

	log.Info().Msg("server stopped")
}

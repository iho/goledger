package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	grpcMiddleware "github.com/iho/goledger/internal/adapter/grpc/middleware"
	pb "github.com/iho/goledger/internal/adapter/grpc/pb/goledger/v1"
	grpcServer "github.com/iho/goledger/internal/adapter/grpc/server"
	httpAdapter "github.com/iho/goledger/internal/adapter/http"
	"github.com/iho/goledger/internal/adapter/http/handler"
	postgresRepo "github.com/iho/goledger/internal/adapter/repository/postgres"
	redisRepo "github.com/iho/goledger/internal/adapter/repository/redis"
	"github.com/iho/goledger/internal/infrastructure/auth"
	"github.com/iho/goledger/internal/infrastructure/config"
	"github.com/iho/goledger/internal/infrastructure/eventpublisher"
	"github.com/iho/goledger/internal/infrastructure/logger"
	"github.com/iho/goledger/internal/infrastructure/postgres"
	"github.com/iho/goledger/internal/infrastructure/redis"
	"github.com/iho/goledger/internal/usecase"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load configuration: " + err.Error())
	}

	// Setup logger
	l := logger.New(logger.Config{
		Level:  cfg.LogLevel,
		Format: "json", // or "text" depending on env
	})
	slog.SetDefault(l)

	ctx := context.Background()

	// Connect to PostgreSQL
	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL, cfg.DatabaseMaxConns, cfg.DatabaseMinConns)
	if err != nil {
		l.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	l.Info("connected to postgres")

	// Run migrations
	if err := postgres.RunMigrations(cfg.DatabaseURL, "internal/infrastructure/postgres/migrations"); err != nil {
		l.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Connect to Redis
	redisClient, err := redis.NewClient(ctx, cfg.RedisURL)
	if err != nil {
		l.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	l.Info("connected to redis")

	// Initialize repositories
	txManager := postgresRepo.NewTxManager(pool)
	accountRepo := postgresRepo.NewAccountRepository(pool)
	transferRepo := postgresRepo.NewTransferRepository(pool)
	entryRepo := postgresRepo.NewEntryRepository(pool)
	ledgerRepo := postgresRepo.NewLedgerRepository(pool)
	holdRepo := postgresRepo.NewHoldRepository(pool)
	outboxRepo := postgresRepo.NewOutboxRepository(pool)
	userRepo := postgresRepo.NewUserRepository(pool)
	idempotencyStore := redisRepo.NewIdempotencyStore(redisClient)
	idGen := postgresRepo.NewULIDGenerator()

	// Initialize use cases with retry support
	retrier := postgresRepo.NewRetrier()
	accountUC := usecase.NewAccountUseCase(accountRepo, idGen)
	transferUC := usecase.NewTransferUseCase(txManager, accountRepo, transferRepo, entryRepo, outboxRepo, idGen).
		WithRetrier(retrier)
	entryUC := usecase.NewEntryUseCase(entryRepo)
	ledgerUC := usecase.NewLedgerUseCase(ledgerRepo)
	holdUC := usecase.NewHoldUseCase(txManager, accountRepo, holdRepo, transferRepo, entryRepo, outboxRepo, idGen)
	userUC := usecase.NewUserUseCase(userRepo)

	// Initialize handlers
	accountHandler := handler.NewAccountHandler(accountUC)
	transferHandler := handler.NewTransferHandler(transferUC)
	entryHandler := handler.NewEntryHandler(entryUC)
	ledgerHandler := handler.NewLedgerHandler(ledgerUC)
	holdHandler := handler.NewHoldHandler(holdUC)
	healthHandler := handler.NewHealthHandler(pool, redisClient)

	// Create JWT manager for authentication
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTExpiration)
	authHandler := handler.NewAuthHandler(jwtManager, userUC)

	// Create router
	router := httpAdapter.NewRouter(httpAdapter.RouterConfig{
		AccountHandler:   accountHandler,
		TransferHandler:  transferHandler,
		EntryHandler:     entryHandler,
		HealthHandler:    healthHandler,
		LedgerHandler:    ledgerHandler,
		HoldHandler:      holdHandler,
		AuthHandler:      authHandler,
		IdempotencyStore: idempotencyStore,
		Logger:           l,
	})

	// Create event publisher worker
	eventPublisher := eventpublisher.NewEventPublisher(eventpublisher.Config{
		OutboxRepo: outboxRepo,
		Publisher:  eventpublisher.NewLogPublisher(l),
		Logger:     l,
	})

	// Start event publisher in background
	publisherCtx, cancelPublisher := context.WithCancel(context.Background())
	defer cancelPublisher()

	go func() {
		if err := eventPublisher.Start(publisherCtx); err != nil && !errors.Is(err, context.Canceled) {
			l.Error("event publisher stopped with error", "error", err)
		}
	}()

	// Create HTTP server with timeouts
	httpServer := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      router,
		ReadTimeout:  cfg.HTTPReadTimeout,
		WriteTimeout: cfg.HTTPWriteTimeout,
		IdleTimeout:  cfg.HTTPIdleTimeout,
	}

	// Create gRPC server with idempotency interceptor
	grpcSrv := grpc.NewServer(
		grpc.UnaryInterceptor(grpcMiddleware.IdempotencyInterceptor(idempotencyStore)),
	)

	// Register gRPC services
	pb.RegisterAccountServiceServer(grpcSrv, grpcServer.NewAccountServer(accountUC))
	pb.RegisterTransferServiceServer(grpcSrv, grpcServer.NewTransferServer(transferUC))
	pb.RegisterHoldServiceServer(grpcSrv, grpcServer.NewHoldServer(holdUC))

	// Register reflection service for grpcurl
	reflection.Register(grpcSrv)

	// Start HTTP server in goroutine
	go func() {
		l.Info("starting HTTP server", "port", cfg.HTTPPort)
		err := httpServer.ListenAndServe()

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			l.Error("HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Start gRPC server in goroutine
	grpcPort := "50051" // Default gRPC port
	if os.Getenv("GRPC_PORT") != "" {
		grpcPort = os.Getenv("GRPC_PORT")
	}

	go func() {
		lis, err := net.Listen("tcp", ":"+grpcPort)
		if err != nil {
			l.Error("failed to listen for gRPC", "error", err)
			os.Exit(1)
		}

		l.Info("starting gRPC server", "port", grpcPort)
		if err := grpcSrv.Serve(lis); err != nil {
			l.Error("gRPC server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	l.Info("shutting down server...")

	// Graceful shutdown with configured timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTPShutdownTimeout)
	defer cancel()

	// Stop event publisher first
	cancelPublisher()
	l.Info("event publisher stopped")

	// Shutdown gRPC server
	grpcSrv.GracefulStop()
	l.Info("gRPC server stopped")

	// Shutdown HTTP server
	if err := httpServer.Shutdown(ctx); err != nil {
		l.Error("HTTP server forced to shutdown", "error", err)
		os.Exit(1)
	}

	l.Info("servers stopped")
}

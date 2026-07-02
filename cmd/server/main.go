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

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	grpcMiddleware "github.com/iho/goledger/internal/adapter/grpc/middleware"
	pb "github.com/iho/goledger/internal/adapter/grpc/pb/goledger/v1"
	grpcServer "github.com/iho/goledger/internal/adapter/grpc/server"
	httpAdapter "github.com/iho/goledger/internal/adapter/http"
	"github.com/iho/goledger/internal/adapter/http/handler"
	postgresRepo "github.com/iho/goledger/internal/adapter/repository/postgres"
	redisRepo "github.com/iho/goledger/internal/adapter/repository/redis"
	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/infrastructure/auth"
	"github.com/iho/goledger/internal/infrastructure/config"
	"github.com/iho/goledger/internal/infrastructure/eventpublisher"
	"github.com/iho/goledger/internal/infrastructure/logger"
	"github.com/iho/goledger/internal/infrastructure/metrics"
	"github.com/iho/goledger/internal/infrastructure/postgres"
	"github.com/iho/goledger/internal/infrastructure/reconciliation"
	"github.com/iho/goledger/internal/infrastructure/redis"
	"github.com/iho/goledger/internal/infrastructure/tracing"
	"github.com/iho/goledger/internal/usecase"
)

func main() {
	os.Exit(run())
}

// run contains the server bootstrap and lifecycle. It returns an exit code
// instead of calling os.Exit directly, so every defer registered along the
// way (closing the DB pool, the Redis client, canceling the event publisher
// context, etc.) actually runs before the process exits.
func run() int {
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

	// Setup distributed tracing (no-op unless TRACING_ENABLED=true)
	shutdownTracing, err := tracing.Setup(ctx, tracing.Config{
		Enabled:      cfg.TracingEnabled,
		ServiceName:  "goledger",
		OTLPEndpoint: cfg.OTLPEndpoint,
	})
	if err != nil {
		l.Error("failed to set up tracing", "error", err)
		return 1
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTPShutdownTimeout)
		defer cancel()
		if err := shutdownTracing(shutdownCtx); err != nil {
			l.Error("failed to shut down tracing", "error", err)
		}
	}()

	// Initialize metrics
	m := metrics.New()

	// Connect to PostgreSQL
	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL, cfg.DatabaseMaxConns, cfg.DatabaseMinConns)
	if err != nil {
		l.Error("failed to connect to postgres", "error", err)
		return 1
	}
	defer pool.Close()

	l.Info("connected to postgres")

	// Run migrations
	if err := postgres.RunMigrations(cfg.DatabaseURL, "internal/infrastructure/postgres/migrations"); err != nil {
		l.Error("failed to run migrations", "error", err)
		return 1
	}

	// Connect to Redis
	redisClient, err := redis.NewClient(ctx, cfg.RedisURL)
	if err != nil {
		l.Error("failed to connect to redis", "error", err)
		return 1
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
	auditRepo := postgresRepo.NewAuditRepository(pool)
	userRepo := postgresRepo.NewUserRepository(pool)
	idempotencyStore := redisRepo.NewIdempotencyStore(redisClient)
	idGen := postgresRepo.NewULIDGenerator()

	// Initialize use cases with retry support
	retrier := postgresRepo.NewRetrier()
	accountUC := usecase.NewAccountUseCase(txManager, accountRepo, auditRepo, idGen, m)
	transferUC := usecase.NewTransferUseCase(txManager, accountRepo, transferRepo, entryRepo, outboxRepo, auditRepo, idGen, m).
		WithRetrier(retrier)
	entryUC := usecase.NewEntryUseCase(entryRepo)
	ledgerUC := usecase.NewLedgerUseCase(ledgerRepo)
	holdUC := usecase.NewHoldUseCase(txManager, accountRepo, holdRepo, transferRepo, entryRepo, outboxRepo, auditRepo, idGen, m)
	userUC := usecase.NewUserUseCase(userRepo)
	reconciliationUC := usecase.NewReconciliationUseCase(accountRepo, entryRepo, ledgerRepo)

	// Initialize handlers
	accountHandler := handler.NewAccountHandler(accountUC)
	transferHandler := handler.NewTransferHandler(transferUC)
	entryHandler := handler.NewEntryHandler(entryUC)
	ledgerHandler := handler.NewLedgerHandler(ledgerUC)
	holdHandler := handler.NewHoldHandler(holdUC)
	healthHandler := handler.NewHealthHandler(pool, redisClient)

	// Create JWT manager for authentication
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTExpiration)
	authHandler := handler.NewAuthHandler(jwtManager, userUC).WithAudit(auditRepo, idGen)
	auditHandler := handler.NewAuditHandler(auditRepo)

	// Create router
	router := httpAdapter.NewRouter(httpAdapter.RouterConfig{
		AccountHandler:   accountHandler,
		TransferHandler:  transferHandler,
		EntryHandler:     entryHandler,
		HealthHandler:    healthHandler,
		LedgerHandler:    ledgerHandler,
		HoldHandler:      holdHandler,
		AuthHandler:      authHandler,
		AuditHandler:     auditHandler,
		IdempotencyStore: idempotencyStore,
		Logger:           l,
		JWTManager:       jwtManager,
		AuthEnabled:      cfg.AuthEnabled,
	})

	// Create event publisher worker
	eventPublisher := eventpublisher.NewEventPublisher(eventpublisher.Config{
		OutboxRepo:  outboxRepo,
		Publisher:   eventpublisher.NewLogPublisher(l),
		Logger:      l,
		Metrics:     m,
		MaxAttempts: cfg.OutboxMaxAttempts,
	})

	// Start event publisher in background
	publisherCtx, cancelPublisher := context.WithCancel(context.Background())
	defer cancelPublisher()

	go func() {
		if err := eventPublisher.Start(publisherCtx); err != nil && !errors.Is(err, context.Canceled) {
			l.Error("event publisher stopped with error", "error", err)
		}
	}()

	// Start scheduled reconciliation in background (0 interval disables it;
	// the on-demand /api/v1/ledger/consistency endpoint keeps working either way)
	var cancelReconciliation context.CancelFunc
	if cfg.ReconciliationInterval > 0 {
		reconciliationScheduler := reconciliation.NewScheduler(reconciliation.Config{
			ReconciliationUC: reconciliationUC,
			Logger:           l,
			Metrics:          m,
			Interval:         cfg.ReconciliationInterval,
		})

		var reconciliationCtx context.Context
		reconciliationCtx, cancelReconciliation = context.WithCancel(context.Background())

		go func() {
			if err := reconciliationScheduler.Start(reconciliationCtx); err != nil && !errors.Is(err, context.Canceled) {
				l.Error("reconciliation scheduler stopped with error", "error", err)
			}
		}()
	}

	// Create HTTP server with timeouts. otelhttp.NewHandler wraps the whole
	// router with one span per request; a no-op when tracing is disabled.
	httpServer := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      otelhttp.NewHandler(router, "goledger-http"),
		ReadTimeout:  cfg.HTTPReadTimeout,
		WriteTimeout: cfg.HTTPWriteTimeout,
		IdleTimeout:  cfg.HTTPIdleTimeout,
	}

	// Create gRPC server with idempotency and, when AUTH_ENABLED, auth/RBAC interceptors.
	unaryInterceptors := []grpc.UnaryServerInterceptor{
		grpcMiddleware.IdempotencyInterceptor(idempotencyStore),
	}
	if cfg.AuthEnabled {
		unaryInterceptors = append(unaryInterceptors,
			grpcMiddleware.AuthInterceptor(jwtManager),
			grpcMiddleware.MethodRoleInterceptor(grpcMethodRoles),
		)
	}

	grpcSrv := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(unaryInterceptors...),
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
	grpcPort := resolveGRPCPort()
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

	if cancelReconciliation != nil {
		cancelReconciliation()
		l.Info("reconciliation scheduler stopped")
	}

	// Shutdown gRPC server
	grpcSrv.GracefulStop()
	l.Info("gRPC server stopped")

	// Shutdown HTTP server
	if err := httpServer.Shutdown(ctx); err != nil {
		l.Error("HTTP server forced to shutdown", "error", err)
		return 1
	}

	l.Info("servers stopped")

	return 0
}

func resolveGRPCPort() string {
	if port := os.Getenv("GRPC_PORT"); port != "" {
		return port
	}
	return "50051"
}

// grpcMethodRoles maps mutating RPCs to their minimum required role, mirroring
// the HTTP route RBAC matrix (admin manages accounts, operator moves money).
// RPCs not listed here only require a valid authenticated user.
var grpcMethodRoles = map[string]domain.Role{
	"/goledger.v1.AccountService/CreateAccount":        domain.RoleAdmin,
	"/goledger.v1.TransferService/CreateTransfer":      domain.RoleOperator,
	"/goledger.v1.TransferService/CreateBatchTransfer": domain.RoleOperator,
	"/goledger.v1.TransferService/ReverseTransfer":     domain.RoleOperator,
	"/goledger.v1.HoldService/HoldFunds":               domain.RoleOperator,
	"/goledger.v1.HoldService/VoidHold":                domain.RoleOperator,
	"/goledger.v1.HoldService/CaptureHold":             domain.RoleOperator,
}

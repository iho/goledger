package testutil

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/ulid/v2"
	"github.com/shopspring/decimal"

	"github.com/iho/goledger/internal/domain"
	"github.com/iho/goledger/internal/infrastructure/postgres"
	"github.com/iho/goledger/internal/infrastructure/postgres/generated"
)

// TestDB provides isolated test database connections.
type TestDB struct {
	Pool    *pgxpool.Pool
	Queries *generated.Queries
	t       *testing.T
}

// NewTestDB creates a new test database connection.
func NewTestDB(t *testing.T) *TestDB {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://ledger:ledger@localhost:5432/ledger?sslmode=disable"
	}

	// Run migrations
	// Assuming tests are run from project root or subdirectories, we need to find migrations.
	// This is a bit hacky for tests but works for typical setups.
	// Try absolute path if in docker, or relative if local.
	migrationsPath := "internal/infrastructure/postgres/migrations"
	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		// Try relative from tests/integration
		migrationsPath = "../../internal/infrastructure/postgres/migrations"
	}
	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		// Try relative from tests/testutil
		migrationsPath = "../../../internal/infrastructure/postgres/migrations"
	}

	if err := postgres.RunMigrations(dbURL, migrationsPath); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("failed to ping test database: %v", err)
	}

	return &TestDB{
		Pool:    pool,
		Queries: generated.New(pool),
		t:       t,
	}
}

// Cleanup closes the database connection.
func (db *TestDB) Cleanup() {
	db.Pool.Close()
}

// TruncateAll removes all data from tables.
func (db *TestDB) TruncateAll(ctx context.Context) {
	db.t.Helper()

	_, err := db.Pool.Exec(ctx, `
		TRUNCATE TABLE holds CASCADE;
		TRUNCATE TABLE entries CASCADE;
		TRUNCATE TABLE transfers CASCADE;
		TRUNCATE TABLE accounts CASCADE;
	`)
	if err != nil {
		db.t.Fatalf("failed to truncate tables: %v", err)
	}
}

// CreateTestAccount creates a test account with given parameters.
func (db *TestDB) CreateTestAccount(ctx context.Context, name, currency string, allowNegative, allowPositive bool) *domain.Account {
	db.t.Helper()

	now := time.Now().UTC()
	id := ulid.Make().String()

	var balance pgtype.Numeric

	_ = balance.Scan("0")

	ts := pgtype.Timestamptz{Time: now, Valid: true}

	_, err := db.Queries.CreateAccount(ctx, generated.CreateAccountParams{
		ID:                   id,
		Name:                 name,
		Currency:             currency,
		Balance:              balance,
		Version:              0,
		AllowNegativeBalance: allowNegative,
		AllowPositiveBalance: allowPositive,
		CreatedAt:            ts,
		UpdatedAt:            ts,
	})
	if err != nil {
		db.t.Fatalf("failed to create test account: %v", err)
	}

	return &domain.Account{
		ID:                   id,
		Name:                 name,
		Currency:             currency,
		Balance:              decimal.Zero,
		Version:              0,
		AllowNegativeBalance: allowNegative,
		AllowPositiveBalance: allowPositive,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}

// CreateTestAccountWithBalance creates a test account with initial balance.
func (db *TestDB) CreateTestAccountWithBalance(ctx context.Context, name, currency string, balance decimal.Decimal, allowNegative, allowPositive bool) *domain.Account {
	db.t.Helper()

	now := time.Now().UTC()
	id := ulid.Make().String()

	var numericBalance pgtype.Numeric

	_ = numericBalance.Scan(balance.String())

	ts := pgtype.Timestamptz{Time: now, Valid: true}

	_, err := db.Queries.CreateAccount(ctx, generated.CreateAccountParams{
		ID:                   id,
		Name:                 name,
		Currency:             currency,
		Balance:              numericBalance,
		Version:              0,
		AllowNegativeBalance: allowNegative,
		AllowPositiveBalance: allowPositive,
		CreatedAt:            ts,
		UpdatedAt:            ts,
	})
	if err != nil {
		db.t.Fatalf("failed to create test account: %v", err)
	}

	return &domain.Account{
		ID:                   id,
		Name:                 name,
		Currency:             currency,
		Balance:              balance,
		Version:              0,
		AllowNegativeBalance: allowNegative,
		AllowPositiveBalance: allowPositive,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}

// GenerateID generates a new ULID.
func GenerateID() string {
	return ulid.Make().String()
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"

	"github.com/iho/goledger/internal/adapter/repository/postgres"
	"github.com/iho/goledger/internal/domain"
	infraPg "github.com/iho/goledger/internal/infrastructure/postgres"
)

var (
	baseURL     string
	timeout     time.Duration
	databaseURL string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "goledger-cli",
		Short: "GoLedger CLI tool",
		Long:  `A command line interface for managing GoLedger.`,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&baseURL, "url", "http://localhost:8080", "Base URL of the GoLedger API")
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 10*time.Second, "Request timeout")
	rootCmd.PersistentFlags().StringVar(&databaseURL, "database-url", os.Getenv("DATABASE_URL"), "Database connection URL")

	// Add commands
	rootCmd.AddCommand(setupCmd())
	rootCmd.AddCommand(migrateCmd())
	rootCmd.AddCommand(userCmd())
	rootCmd.AddCommand(ledgerCmd())
	rootCmd.AddCommand(hashPasswordCmd())
	rootCmd.AddCommand(serveCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// ============ SETUP COMMAND ============

func setupCmd() *cobra.Command {
	var adminEmail, adminPassword, adminName string

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Initialize the system (migrate + create admin)",
		Long:  `Run database migrations and create the initial admin user.`,
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()

			if databaseURL == "" {
				fmt.Println("❌ Error: DATABASE_URL is required")
				fmt.Println("   Set it via --database-url flag or DATABASE_URL environment variable")
				os.Exit(1)
			}

			// Connect to database
			fmt.Println("🔗 Connecting to database...")
			pool, err := pgxpool.New(ctx, databaseURL)
			if err != nil {
				fmt.Printf("❌ Failed to connect: %v\n", err)
				os.Exit(1)
			}
			defer pool.Close()
			fmt.Println("✅ Connected to database")

			// Run migrations
			fmt.Println("📦 Running migrations...")
			if err := infraPg.RunMigrations(databaseURL, "internal/infrastructure/postgres/migrations"); err != nil {
				fmt.Printf("❌ Migration failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✅ Migrations applied")

			// Create admin user
			fmt.Println("👤 Creating admin user...")
			if err := createUser(ctx, pool, adminEmail, adminName, adminPassword, "admin"); err != nil {
				fmt.Printf("⚠️  Admin creation: %v\n", err)
			} else {
				fmt.Printf("✅ Admin user created: %s\n", adminEmail)
			}

			fmt.Println("\n🎉 Setup complete!")
			fmt.Printf("\n📧 Login: %s\n🔑 Password: %s\n", adminEmail, adminPassword)
		},
	}

	cmd.Flags().StringVar(&adminEmail, "admin-email", "admin@goledger.io", "Admin email")
	cmd.Flags().StringVar(&adminPassword, "admin-password", "Admin123", "Admin password")
	cmd.Flags().StringVar(&adminName, "admin-name", "Admin User", "Admin name")

	return cmd
}

// ============ MIGRATE COMMAND ============

func migrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Database migration commands",
	}

	upCmd := &cobra.Command{
		Use:   "up",
		Short: "Apply all pending migrations",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			pool := mustConnectDB(ctx)
			defer pool.Close()

			if err := infraPg.RunMigrations(databaseURL, "internal/infrastructure/postgres/migrations"); err != nil {
				fmt.Printf("❌ Migration failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✅ Migrations applied successfully")
		},
	}

	downCmd := &cobra.Command{
		Use:   "down",
		Short: "Rollback the last migration",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			pool := mustConnectDB(ctx)
			defer pool.Close()

			if err := infraPg.RunMigrationsDown(databaseURL, "internal/infrastructure/postgres/migrations"); err != nil {
				fmt.Printf("❌ Rollback failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✅ Migration rolled back successfully")
		},
	}

	cmd.AddCommand(upCmd, downCmd)
	return cmd
}

// ============ USER COMMAND ============

func userCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "User management commands",
	}

	// Create user
	var email, password, name, role string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new user",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			pool := mustConnectDB(ctx)
			defer pool.Close()

			if err := createUser(ctx, pool, email, name, password, role); err != nil {
				fmt.Printf("❌ Failed to create user: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("✅ User created: %s (role: %s)\n", email, role)
		},
	}
	createCmd.Flags().StringVar(&email, "email", "", "User email (required)")
	createCmd.Flags().StringVar(&password, "password", "", "User password (required)")
	createCmd.Flags().StringVar(&name, "name", "", "User name (required)")
	createCmd.Flags().StringVar(&role, "role", "viewer", "User role (admin, operator, viewer)")
	createCmd.MarkFlagRequired("email")
	createCmd.MarkFlagRequired("password")
	createCmd.MarkFlagRequired("name")

	// List users
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all users",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			pool := mustConnectDB(ctx)
			defer pool.Close()

			userRepo := postgres.NewUserRepository(pool)
			users, err := userRepo.List(ctx, 100, 0)
			if err != nil {
				fmt.Printf("❌ Failed to list users: %v\n", err)
				os.Exit(1)
			}

			if len(users) == 0 {
				fmt.Println("No users found")
				return
			}

			fmt.Printf("%-36s %-30s %-15s %-10s %-8s\n", "ID", "EMAIL", "NAME", "ROLE", "ACTIVE")
			fmt.Println("------------------------------------------------------------------------------------------------")
			for _, u := range users {
				active := "yes"
				if !u.Active {
					active = "no"
				}
				fmt.Printf("%-36s %-30s %-15s %-10s %-8s\n", u.ID, u.Email, truncate(u.Name, 15), u.Role, active)
			}
		},
	}

	cmd.AddCommand(createCmd, listCmd)
	return cmd
}

// ============ LEDGER COMMAND ============

func ledgerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ledger",
		Short: "Ledger operations",
	}

	consistencyCmd := &cobra.Command{
		Use:   "consistency",
		Short: "Check ledger consistency",
		Run: func(cmd *cobra.Command, args []string) {
			checkConsistency()
		},
	}

	cmd.AddCommand(consistencyCmd)
	return cmd
}

// ============ SERVE COMMAND ============

func serveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP/gRPC server",
		Long:  `Start the GoLedger HTTP and gRPC server. Uses environment variables for configuration.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("🚀 Starting GoLedger server...")
			fmt.Println("   Use 'go run ./cmd/server' or the compiled server binary")
			fmt.Println("   Environment variables:")
			fmt.Println("     DATABASE_URL  - PostgreSQL connection string")
			fmt.Println("     REDIS_URL     - Redis connection string")
			fmt.Println("     HTTP_PORT     - HTTP port (default: 8080)")
			fmt.Println("     GRPC_PORT     - gRPC port (default: 50051)")
			fmt.Println("     JWT_SECRET    - JWT signing secret")
			fmt.Println("     AUTH_ENABLED  - Enable authentication (default: false)")
		},
	}
}

// ============ HASH PASSWORD COMMAND ============

func hashPasswordCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hash-password [password]",
		Short: "Generate bcrypt hash for a password",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			hash, err := bcrypt.GenerateFromPassword([]byte(args[0]), bcrypt.DefaultCost)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(string(hash))
		},
	}
}

// ============ HELPERS ============

func mustConnectDB(ctx context.Context) *pgxpool.Pool {
	if databaseURL == "" {
		fmt.Println("❌ Error: DATABASE_URL is required")
		fmt.Println("   Set via --database-url or DATABASE_URL env var")
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		fmt.Printf("❌ Failed to connect to database: %v\n", err)
		os.Exit(1)
	}

	if err := pool.Ping(ctx); err != nil {
		fmt.Printf("❌ Failed to ping database: %v\n", err)
		os.Exit(1)
	}

	return pool
}

func createUser(ctx context.Context, pool *pgxpool.Pool, email, name, password, role string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	userRepo := postgres.NewUserRepository(pool)
	idGen := postgres.NewULIDGenerator()

	user := &domain.User{
		ID:             idGen.Generate(),
		Email:          email,
		Name:           name,
		HashedPassword: string(hashedPassword),
		Role:           domain.Role(role),
		Active:         true,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	return userRepo.Create(ctx, user)
}

func checkConsistency() {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(baseURL + "/api/v1/ledger/consistency")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("❌ Consistency check FAILED (Status: %d)\n%s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("Failed to parse response: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Consistency check PASSED")
	fmt.Printf("Status: %s\n", result["status"])
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"

	"github.com/iho/goledger/internal/adapter/repository/postgres"
	"github.com/iho/goledger/internal/domain"
	infraPostgres "github.com/iho/goledger/internal/infrastructure/postgres"
	"github.com/iho/goledger/internal/usecase"
)

var (
	databaseURL string
	jsonOutput  bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "goledger-cli",
		Short: "GoLedger CLI tool",
		Long:  `A command line interface for managing GoLedger.`,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&databaseURL, "database-url", os.Getenv("DATABASE_URL"), "Database connection URL")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	// Add commands
	rootCmd.AddCommand(setupCmd())
	rootCmd.AddCommand(migrateCmd())
	rootCmd.AddCommand(userCmd())
	rootCmd.AddCommand(accountCmd())
	rootCmd.AddCommand(transferCmd())
	rootCmd.AddCommand(holdCmd())
	rootCmd.AddCommand(ledgerCmd())
	rootCmd.AddCommand(hashPasswordCmd())

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
			if err := infraPostgres.RunMigrations(databaseURL, "internal/infrastructure/postgres/migrations"); err != nil {
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

			if err := infraPostgres.RunMigrations(databaseURL, "internal/infrastructure/postgres/migrations"); err != nil {
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

			if err := infraPostgres.RunMigrationsDown(databaseURL, "internal/infrastructure/postgres/migrations"); err != nil {
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

// ============ ACCOUNT COMMAND ============

func accountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account",
		Short: "Account management",
	}

	// Create account
	var name, currency string
	var allowNegative, allowPositive bool
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new account",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			pool := mustConnectDB(ctx)
			defer pool.Close()

			accountUC := usecase.NewAccountUseCase(
				postgres.NewAccountRepository(pool),
				postgres.NewULIDGenerator(),
			)

			account, err := accountUC.CreateAccount(ctx, usecase.CreateAccountInput{
				Name:                 name,
				Currency:             currency,
				AllowNegativeBalance: allowNegative,
				AllowPositiveBalance: allowPositive,
			})
			if err != nil {
				fmt.Printf("❌ Failed to create account: %v\n", err)
				os.Exit(1)
			}

			if jsonOutput {
				printJSON(account)
			} else {
				fmt.Printf("✅ Account created: %s\n", account.ID)
				fmt.Printf("   Name: %s\n", account.Name)
				fmt.Printf("   Currency: %s\n", account.Currency)
			}
		},
	}
	createCmd.Flags().StringVar(&name, "name", "", "Account name (required)")
	createCmd.Flags().StringVar(&currency, "currency", "USD", "Currency code")
	createCmd.Flags().BoolVar(&allowNegative, "allow-negative", false, "Allow negative balance")
	createCmd.Flags().BoolVar(&allowPositive, "allow-positive", true, "Allow positive balance")
	createCmd.MarkFlagRequired("name")

	// List accounts
	var limit, offset int
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all accounts",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			pool := mustConnectDB(ctx)
			defer pool.Close()

			accountUC := usecase.NewAccountUseCase(
				postgres.NewAccountRepository(pool),
				postgres.NewULIDGenerator(),
			)

			accounts, err := accountUC.ListAccounts(ctx, usecase.ListAccountsInput{
				Limit:  limit,
				Offset: offset,
			})
			if err != nil {
				fmt.Printf("❌ Failed to list accounts: %v\n", err)
				os.Exit(1)
			}

			if jsonOutput {
				printJSON(accounts)
			} else {
				fmt.Printf("%-28s %-20s %-8s %-15s\n", "ID", "NAME", "CURRENCY", "BALANCE")
				fmt.Println("-----------------------------------------------------------------------")
				for _, a := range accounts {
					fmt.Printf("%-28s %-20s %-8s %-15s\n", a.ID, truncate(a.Name, 20), a.Currency, a.Balance.String())
				}
			}
		},
	}
	listCmd.Flags().IntVar(&limit, "limit", 100, "Limit results")
	listCmd.Flags().IntVar(&offset, "offset", 0, "Offset results")

	// Get account
	getCmd := &cobra.Command{
		Use:   "get [id]",
		Short: "Get account by ID",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			pool := mustConnectDB(ctx)
			defer pool.Close()

			accountUC := usecase.NewAccountUseCase(
				postgres.NewAccountRepository(pool),
				postgres.NewULIDGenerator(),
			)

			account, err := accountUC.GetAccount(ctx, args[0])
			if err != nil {
				fmt.Printf("❌ Account not found: %v\n", err)
				os.Exit(1)
			}

			if jsonOutput {
				printJSON(account)
			} else {
				fmt.Printf("ID:       %s\n", account.ID)
				fmt.Printf("Name:     %s\n", account.Name)
				fmt.Printf("Currency: %s\n", account.Currency)
				fmt.Printf("Balance:  %s\n", account.Balance.String())
				fmt.Printf("Version:  %d\n", account.Version)
			}
		},
	}

	cmd.AddCommand(createCmd, listCmd, getCmd)
	return cmd
}

// ============ TRANSFER COMMAND ============

func transferCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer",
		Short: "Transfer management",
	}

	// Create transfer
	var fromID, toID, amount, description string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new transfer",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			pool := mustConnectDB(ctx)
			defer pool.Close()

			txManager := postgres.NewTxManager(pool)
			transferUC := usecase.NewTransferUseCase(
				txManager,
				postgres.NewAccountRepository(pool),
				postgres.NewTransferRepository(pool),
				postgres.NewEntryRepository(pool),
				postgres.NewOutboxRepository(pool),
				postgres.NewULIDGenerator(),
			)

			amt, err := decimal.NewFromString(amount)
			if err != nil {
				fmt.Printf("❌ Invalid amount: %v\n", err)
				os.Exit(1)
			}

			transfer, err := transferUC.CreateTransfer(ctx, usecase.CreateTransferInput{
				FromAccountID: fromID,
				ToAccountID:   toID,
				Amount:        amt,
			})
			if err != nil {
				fmt.Printf("❌ Failed to create transfer: %v\n", err)
				os.Exit(1)
			}

			if jsonOutput {
				printJSON(transfer)
			} else {
				fmt.Printf("✅ Transfer created: %s\n", transfer.ID)
				fmt.Printf("   From: %s\n", transfer.FromAccountID)
				fmt.Printf("   To:   %s\n", transfer.ToAccountID)
				fmt.Printf("   Amount: %s\n", transfer.Amount.String())
			}
		},
	}
	createCmd.Flags().StringVar(&fromID, "from", "", "Source account ID (required)")
	createCmd.Flags().StringVar(&toID, "to", "", "Destination account ID (required)")
	createCmd.Flags().StringVar(&amount, "amount", "", "Transfer amount (required)")
	createCmd.Flags().StringVar(&description, "description", "", "Transfer description")
	createCmd.MarkFlagRequired("from")
	createCmd.MarkFlagRequired("to")
	createCmd.MarkFlagRequired("amount")

	// Get transfer
	getCmd := &cobra.Command{
		Use:   "get [id]",
		Short: "Get transfer by ID",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			pool := mustConnectDB(ctx)
			defer pool.Close()

			txManager := postgres.NewTxManager(pool)
			transferUC := usecase.NewTransferUseCase(
				txManager,
				postgres.NewAccountRepository(pool),
				postgres.NewTransferRepository(pool),
				postgres.NewEntryRepository(pool),
				postgres.NewOutboxRepository(pool),
				postgres.NewULIDGenerator(),
			)

			transfer, err := transferUC.GetTransfer(ctx, args[0])
			if err != nil {
				fmt.Printf("❌ Transfer not found: %v\n", err)
				os.Exit(1)
			}

			if jsonOutput {
				printJSON(transfer)
			} else {
				fmt.Printf("ID:     %s\n", transfer.ID)
				fmt.Printf("From:   %s\n", transfer.FromAccountID)
				fmt.Printf("To:     %s\n", transfer.ToAccountID)
				fmt.Printf("Amount: %s\n", transfer.Amount.String())
			}
		},
	}

	cmd.AddCommand(createCmd, getCmd)
	return cmd
}

// ============ HOLD COMMAND ============

func holdCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hold",
		Short: "Hold management",
	}

	// Create hold
	var accountID, amount, description string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new hold",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			pool := mustConnectDB(ctx)
			defer pool.Close()

			txManager := postgres.NewTxManager(pool)
			holdUC := usecase.NewHoldUseCase(
				txManager,
				postgres.NewAccountRepository(pool),
				postgres.NewHoldRepository(pool),
				postgres.NewTransferRepository(pool),
				postgres.NewEntryRepository(pool),
				postgres.NewOutboxRepository(pool),
				postgres.NewULIDGenerator(),
			)

			amt, err := decimal.NewFromString(amount)
			if err != nil {
				fmt.Printf("❌ Invalid amount: %v\n", err)
				os.Exit(1)
			}

			hold, err := holdUC.HoldFunds(ctx, accountID, amt)
			if err != nil {
				fmt.Printf("❌ Failed to create hold: %v\n", err)
				os.Exit(1)
			}

			if jsonOutput {
				printJSON(hold)
			} else {
				fmt.Printf("✅ Hold created: %s\n", hold.ID)
				fmt.Printf("   Account: %s\n", hold.AccountID)
				fmt.Printf("   Amount: %s\n", hold.Amount.String())
			}
		},
	}
	createCmd.Flags().StringVar(&accountID, "account", "", "Account ID (required)")
	createCmd.Flags().StringVar(&amount, "amount", "", "Hold amount (required)")
	createCmd.Flags().StringVar(&description, "description", "", "Hold description")
	createCmd.MarkFlagRequired("account")
	createCmd.MarkFlagRequired("amount")

	// Capture hold
	var captureToID string
	captureCmd := &cobra.Command{
		Use:   "capture [hold-id]",
		Short: "Capture a hold (execute the transfer)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			pool := mustConnectDB(ctx)
			defer pool.Close()

			txManager := postgres.NewTxManager(pool)
			holdUC := usecase.NewHoldUseCase(
				txManager,
				postgres.NewAccountRepository(pool),
				postgres.NewHoldRepository(pool),
				postgres.NewTransferRepository(pool),
				postgres.NewEntryRepository(pool),
				postgres.NewOutboxRepository(pool),
				postgres.NewULIDGenerator(),
			)

			transfer, err := holdUC.CaptureHold(ctx, args[0], captureToID)
			if err != nil {
				fmt.Printf("❌ Failed to capture hold: %v\n", err)
				os.Exit(1)
			}

			if jsonOutput {
				printJSON(transfer)
			} else {
				fmt.Printf("✅ Hold captured, transfer created: %s\n", transfer.ID)
			}
		},
	}
	captureCmd.Flags().StringVar(&captureToID, "to", "", "Destination account ID (required)")
	captureCmd.MarkFlagRequired("to")

	// Void hold
	voidCmd := &cobra.Command{
		Use:   "void [hold-id]",
		Short: "Void a hold (cancel it)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			pool := mustConnectDB(ctx)
			defer pool.Close()

			txManager := postgres.NewTxManager(pool)
			holdUC := usecase.NewHoldUseCase(
				txManager,
				postgres.NewAccountRepository(pool),
				postgres.NewHoldRepository(pool),
				postgres.NewTransferRepository(pool),
				postgres.NewEntryRepository(pool),
				postgres.NewOutboxRepository(pool),
				postgres.NewULIDGenerator(),
			)

			if err := holdUC.VoidHold(ctx, args[0]); err != nil {
				fmt.Printf("❌ Failed to void hold: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("✅ Hold voided: %s\n", args[0])
		},
	}

	cmd.AddCommand(createCmd, captureCmd, voidCmd)
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
			ctx := context.Background()
			pool := mustConnectDB(ctx)
			defer pool.Close()
			checkConsistency(pool)
		},
	}

	cmd.AddCommand(consistencyCmd)
	return cmd
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

func checkConsistency(pool *pgxpool.Pool) {
	ctx := context.Background()
	ledgerRepo := postgres.NewLedgerRepository(pool)
	ledgerUC := usecase.NewLedgerUseCase(ledgerRepo)

	consistent, err := ledgerUC.CheckConsistency(ctx)
	if err != nil {
		fmt.Printf("❌ Consistency check failed: %v\n", err)
		os.Exit(1)
	}

	if consistent {
		fmt.Println("✅ Ledger is consistent")
	} else {
		fmt.Println("❌ Ledger is NOT consistent")
		os.Exit(1)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

func printJSON(v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}

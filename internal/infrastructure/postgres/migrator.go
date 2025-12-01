package postgres

import (
	"errors"
	"fmt"

	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations runs database migrations.
func RunMigrations(databaseURL, migrationsPath string) error {
	m, err := migrate.New(
		"file://"+migrationsPath,
		databaseURL,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("database migrations: no change")
			return nil
		}
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	slog.Info("database migrations: applied successfully")
	return nil
}

// RunMigrationsDown rolls back the last migration.
func RunMigrationsDown(databaseURL, migrationsPath string) error {
	m, err := migrate.New(
		"file://"+migrationsPath,
		databaseURL,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Steps(-1); err != nil {
		return fmt.Errorf("failed to rollback migration: %w", err)
	}

	slog.Info("database migrations: rolled back successfully")
	return nil
}

#!/bin/bash
set -e

# Default values
DB_URL="${DATABASE_URL:-postgres://ledger:ledger@localhost:5432/ledger?sslmode=disable}"
MIGRATIONS_DIR="${MIGRATIONS_DIR:-./internal/infrastructure/postgres/migrations}"

echo "Running migrations from $MIGRATIONS_DIR"

# Apply each SQL file in order
for file in "$MIGRATIONS_DIR"/*.sql; do
    if [ -f "$file" ]; then
        echo "Applying: $(basename "$file")"
        psql "$DB_URL" -f "$file"
    fi
done

echo "Migrations completed successfully"

#!/bin/bash
# Create test users for development/testing

set -e

DB_URL="${DATABASE_URL:-postgres://ledger:ledger@localhost:5432/ledger?sslmode=disable}"

echo "Creating test users..."

# Create admin user
psql "$DB_URL" <<EOF
INSERT INTO users (id, email, hashed_password, role, active, created_at, updated_at)
VALUES (
    'user-admin-1',
    'admin@goledger.io',
    '\$2a\$10\$rKZL3vGKjN7cQQxqJ5FZLOYGZmZ9Z5m9Z5m9Z5m9Z5m9Z5m9Z5m9Z5',  -- Password: Admin123
    'admin',
    true,
    NOW(),
    NOW()
)
ON CONFLICT (email) DO NOTHING;

INSERT INTO users (id, email, hashed_password, role, active, created_at, updated_at)
VALUES (
    'user-operator-1',
    'operator@goledger.io',
    '\$2a\$10\$rKZL3vGKjN7cQQxqJ5FZLOYGZmZ9Z5m9Z5m9Z5m9Z5m9Z5m9Z5m9Z5',  -- Password: Operator123
    'operator',
    true,
    NOW(),
    NOW()
)
ON CONFLICT (email) DO NOTHING;

INSERT INTO users (id, email, hashed_password, role, active, created_at, updated_at)
VALUES (
    'user-viewer-1',
    'viewer@goledger.io',
    '\$2a\$10\$rKZL3vGKjN7cQQxqJ5FZLOYGZmZ9Z5m9Z5m9Z5m9Z5m9Z5m9Z5m9Z5',  -- Password: Viewer123
    'viewer',
    true,
    NOW(),
    NOW()
)
ON CONFLICT (email) DO NOTHING;
EOF

echo "✅ Test users created successfully!"
echo ""
echo "📧 Admin:    admin@goledger.io    / Admin123"
echo "📧 Operator: operator@goledger.io / Operator123"
echo "📧 Viewer:   viewer@goledger.io   / Viewer123"

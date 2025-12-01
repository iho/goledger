#!/bin/bash
set -e

echo "🚀 GoLedger Setup & Test Script"
echo "================================"

DB_URL="postgres://ledger:ledger@localhost:5432/ledger?sslmode=disable"

# 1. Build CLI
echo "1️⃣  Building CLI..."
go build -o bin/cli ./cmd/cli

# 2. Clean everything
echo "2️⃣  Cleaning old containers and volumes..."
docker-compose -f docker-compose.full.yml down -v 2>/dev/null || true

# 3. Start services (will build automatically)
echo "3️⃣  Starting services..."
docker-compose -f docker-compose.full.yml up -d --build

# 4. Wait for PostgreSQL
echo "4️⃣  Waiting for PostgreSQL to be ready..."
for i in {1..30}; do
    if docker exec goledger-postgres pg_isready -U ledger >/dev/null 2>&1; then
        echo "✅ PostgreSQL is ready!"
        break
    fi
    echo "   Waiting... ($i/30)"
    sleep 1
done

# 5. Run setup via CLI (migrations + admin user)
echo "5️⃣  Running setup via CLI..."
./bin/cli setup --database-url "$DB_URL" || echo "   Setup may have partially completed"

# 6. Create additional test users via CLI
echo "6️⃣  Creating test users via CLI..."
./bin/cli user create --email "operator@goledger.io" --name "Operator User" --password "Operator123" --role operator --database-url "$DB_URL" 2>/dev/null || echo "   Operator user exists"
./bin/cli user create --email "viewer@goledger.io" --name "Viewer User" --password "Viewer123" --role viewer --database-url "$DB_URL" 2>/dev/null || echo "   Viewer user exists"

# 7. List users
echo "7️⃣  Listing users..."
./bin/cli user list --database-url "$DB_URL"

# 8. Wait for app to be ready
echo "8️⃣  Waiting for application to be ready..."
for i in {1..30}; do
    if curl -s http://localhost:8080/health >/dev/null 2>&1; then
        echo "✅ Application is ready!"
        break
    fi
    echo "   Waiting... ($i/30)"
    sleep 1
done

# 9. Test authentication
echo "9️⃣  Testing authentication..."
RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@goledger.io","password":"Admin123"}')

if echo "$RESPONSE" | jq -e '.token' >/dev/null 2>&1; then
    echo "✅ Authentication working!"
    TOKEN=$(echo "$RESPONSE" | jq -r '.token')
    echo "   Token: ${TOKEN:0:50}..."
else
    echo "❌ Authentication failed!"
    echo "   Response: $RESPONSE"
    exit 1
fi

# 10. Show service status
echo ""
echo "🎉 Setup complete! Services running:"
echo "================================"
echo "API:        http://localhost:8080"
echo "Metrics:    http://localhost:8080/metrics"
echo "Health:     http://localhost:8080/health"
echo "Grafana:    http://localhost:3000 (admin/admin)"
echo "Prometheus: http://localhost:9090"
echo "PgAdmin:    http://localhost:5050 (admin@goledger.io/admin)"
echo ""
echo "Test credentials:"
echo "  Admin:    admin@goledger.io / Admin123"
echo "  Operator: operator@goledger.io / Operator123"
echo "  Viewer:   viewer@goledger.io / Viewer123"

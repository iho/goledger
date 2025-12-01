#!/bin/bash
set -e

echo "🚀 GoLedger Setup & Test Script"
echo "================================"

# 1. Clean everything
echo "1️⃣  Cleaning old containers and volumes..."
docker-compose -f docker-compose.full.yml down -v 2>/dev/null || true

# 2. Start services (will build automatically)
echo "4️⃣  Starting services..."
docker-compose -f docker-compose.full.yml up -d

# 5. Wait for PostgreSQL
echo "5️⃣  Waiting for PostgreSQL to be ready..."
for i in {1..30}; do
    if docker exec goledger-postgres pg_isready -U ledger >/dev/null 2>&1; then
        echo "✅ PostgreSQL is ready!"
        break
    fi
    echo "   Waiting... ($i/30)"
    sleep 1
done

# 6. Run migrations
echo "6️⃣  Running database migrations..."
docker exec -i goledger-postgres psql -U ledger -d ledger < migrations/000006_create_audit_logs.up.sql 2>/dev/null || echo "   Migration 6 already applied"
docker exec -i goledger-postgres psql -U ledger -d ledger < migrations/000007_create_users.up.sql 2>/dev/null || echo "   Migration 7 already applied"

# 7. Create test users
echo "7️⃣  Creating test users..."
docker exec -i goledger-postgres psql -U ledger -d ledger < scripts/create_test_users.sql 2>/dev/null || echo "   Users already exist"

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

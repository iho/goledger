# GoLedger

A double-entry ledger implementation in Go with PostgreSQL and Redis.

## Features

- **Double-entry accounting** - Every transfer creates balanced debit/credit entries
- **Clean Architecture** - Domain, Use Cases, Adapters, Infrastructure layers
- **Type-safe SQL** - Generated with sqlc
- **Idempotency** - Redis-backed request deduplication
- **Concurrent-safe** - Deadlock prevention via sorted account locking

## Quick Start

```bash
# Start services
docker-compose up -d

# Run the server
go run ./cmd/server

# Or build and run
make build
./bin/goledger
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/accounts` | Create account |
| GET | `/api/v1/accounts/:id` | Get account |
| GET | `/api/v1/accounts` | List accounts |
| POST | `/api/v1/transfers` | Create transfer |
| POST | `/api/v1/transfers/batch` | Batch transfer (atomic) |
| GET | `/api/v1/transfers/:id` | Get transfer |
| GET | `/api/v1/accounts/:id/entries` | List entries |
| GET | `/health` | Health check |
| GET | `/ready` | Readiness check |

## Example Usage

```bash
# Create accounts
curl -X POST http://localhost:8080/api/v1/accounts \
  -H "Content-Type: application/json" \
  -d '{"name": "alice", "currency": "USD", "allow_negative_balance": false, "allow_positive_balance": true}'

curl -X POST http://localhost:8080/api/v1/accounts \
  -H "Content-Type: application/json" \
  -d '{"name": "bob", "currency": "USD", "allow_negative_balance": false, "allow_positive_balance": true}'

# Create transfer
curl -X POST http://localhost:8080/api/v1/transfers \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: unique-key-123" \
  -d '{"from_account_id": "ALICE_ID", "to_account_id": "BOB_ID", "amount": "100.50"}'
```

## Development

```bash
make deps          # Download dependencies
make sqlc          # Generate SQL code
make build         # Build binary
make test          # Run unit tests
make docker-up     # Start Postgres + Redis
make docker-down   # Stop services
```

## License

MIT
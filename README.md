# GoLedger

A production-ready double-entry ledger implementation in Go with PostgreSQL and Redis.

## Features

- **Double-entry accounting** - Every transfer creates balanced debit/credit entries
- **Clean Architecture** - Domain, Use Cases, Adapters, Infrastructure layers
- **Type-safe SQL** - Generated with sqlc
- **Idempotency** - Redis-backed request deduplication
- **Concurrent-safe** - Deadlock prevention via sorted account locking
- **Observability** - Prometheus metrics, structured logging (zerolog)
- **Production-ready** - Health checks, graceful shutdown, configurable timeouts

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    HTTP Layer                            │
│  (chi router, middleware, handlers)                      │
└───────────────────────┬─────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────┐
│                   Use Cases                              │
│  (AccountUseCase, TransferUseCase, EntryUseCase)        │
└───────────────────────┬─────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────┐
│                    Domain                                │
│  (Account, Transfer, Entry - business rules)            │
└───────────────────────┬─────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────┐
│               Infrastructure                             │
│  (PostgreSQL, Redis, sqlc generated code)               │
└─────────────────────────────────────────────────────────┘
```

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
| GET | `/api/v1/accounts/:id/balance/history` | Historical balance |
| GET | `/health` | Liveness probe |
| GET | `/ready` | Readiness probe |
| GET | `/metrics` | Prometheus metrics |

## Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `DATABASE_URL` | `postgres://ledger:ledger@localhost:5432/ledger?sslmode=disable` | PostgreSQL URL |
| `DATABASE_MAX_CONNS` | `25` | Max pool connections |
| `DATABASE_MIN_CONNS` | `5` | Min pool connections |
| `REDIS_URL` | `redis://localhost:6379` | Redis URL |
| `HTTP_PORT` | `8080` | HTTP server port |
| `HTTP_READ_TIMEOUT` | `30s` | Request read timeout |
| `HTTP_WRITE_TIMEOUT` | `30s` | Response write timeout |
| `HTTP_IDLE_TIMEOUT` | `60s` | Keep-alive timeout |
| `HTTP_SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown timeout |
| `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `IDEMPOTENCY_TTL` | `24h` | Idempotency key TTL |

## Example Usage

```bash
# Create accounts
curl -X POST http://localhost:8080/api/v1/accounts \
  -H "Content-Type: application/json" \
  -d '{"name": "alice", "currency": "USD", "allow_negative_balance": false, "allow_positive_balance": true}'

curl -X POST http://localhost:8080/api/v1/accounts \
  -H "Content-Type: application/json" \
  -d '{"name": "bob", "currency": "USD", "allow_negative_balance": false, "allow_positive_balance": true}'

# Create transfer (with idempotency key)
curl -X POST http://localhost:8080/api/v1/transfers \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: unique-key-123" \
  -d '{"from_account_id": "ALICE_ID", "to_account_id": "BOB_ID", "amount": "100.50"}'

# Batch transfer (atomic)
curl -X POST http://localhost:8080/api/v1/transfers/batch \
  -H "Content-Type: application/json" \
  -d '{"transfers": [
    {"from_account_id": "A", "to_account_id": "B", "amount": "50"},
    {"from_account_id": "B", "to_account_id": "C", "amount": "25"}
  ]}'
```

## Development

```bash
make deps           # Download dependencies
make generate       # Generate sqlc + mocks
make build          # Build binary
make test           # Run unit tests (gotestsum)
make test-all       # Run all tests with race detector
make test-coverage  # Generate coverage report
make lint           # Run golangci-lint
make docker-up      # Start Postgres + Redis
make docker-down    # Stop services
```

## Testing

```bash
# Unit tests
make test

# Integration tests (requires Docker)
make test-integration

# All tests with coverage
make test-coverage
```

## Tech Stack

- **Go 1.23+** - Language
- **PostgreSQL 16** - Primary database
- **Redis 7** - Idempotency store
- **chi** - HTTP router
- **sqlc** - Type-safe SQL
- **pgx/v5** - PostgreSQL driver
- **zerolog** - Structured logging
- **gomock** - Mock generation
- **golangci-lint** - Linting
- **gotestsum** - Test runner

## License

MIT
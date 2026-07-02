# GoLedger

![Go Version](https://img.shields.io/badge/go-1.24+-00ADD8.svg)
![License](https://img.shields.io/badge/license-GPLv3-blue.svg)

A production-ready double-entry ledger implementation in Go with PostgreSQL and Redis.

## Features

- **Double-entry accounting** - Every transfer creates balanced debit/credit entries
- **Clean Architecture** - Domain, Use Cases, Adapters, Infrastructure layers
- **Type-safe SQL** - Generated with sqlc
- **Idempotency** - Redis-backed request deduplication
- **Concurrent-safe** - Deadlock prevention via sorted account locking
- **Observability** - Prometheus metrics, structured logging (slog)
- **Production-ready** - Health checks, graceful shutdown, configurable timeouts
- **CLI Tool** - Comprehensive command-line interface for management and setup

## Quick Start

### Automated Setup
The easiest way to get started is using the setup script, which handles building, container startup, migrations, and user creation:

```bash
./scripts/setup-and-test.sh
```

### Manual Setup

1. **Start Infrastructure**
   ```bash
   docker-compose -f docker-compose.full.yml up -d
   ```

2. **Build CLI**
   ```bash
   go build -o bin/cli ./cmd/cli
   ```

3. **Run Setup (Migrations + Admin)**
   ```bash
   export DATABASE_URL="postgres://ledger:ledger@localhost:5432/ledger?sslmode=disable"
   ./bin/cli setup --database-url "$DATABASE_URL"
   ```

## CLI Tool

GoLedger comes with a powerful CLI for managing the system.

```bash
# Build the CLI
go build -o bin/cli ./cmd/cli

# Set DB URL (or use --database-url flag)
export DATABASE_URL="postgres://ledger:ledger@localhost:5432/ledger?sslmode=disable"
```

### Common Commands

| Command | Description | Example |
|---------|-------------|---------|
| `user create` | Create a new user | `./bin/cli user create --email u@x.com --password pass --role admin` |
| `user list` | List users | `./bin/cli user list` |
| `account create` | Create an account | `./bin/cli account create --name "Wallet" --currency USD` |
| `account list` | List accounts | `./bin/cli account list` |
| `account get [id]` | Get an account | `./bin/cli account get acc_123` |
| `transfer create` | Transfer funds | `./bin/cli transfer create --from [id] --to [id] --amount 100` |
| `transfer get [id]` | Get a transfer | `./bin/cli transfer get txn_123` |
| `hold create` | Hold funds | `./bin/cli hold create --account [id] --amount 50` |
| `hold capture [hold-id]` | Capture a hold | `./bin/cli hold capture hold_123 --to acc_456` |
| `hold void [hold-id]` | Void a hold | `./bin/cli hold void hold_123` |
| `ledger consistency` | Check ledger consistency | `./bin/cli ledger consistency` |
| `hash-password [password]` | Hash a password for manual DB insertion | `./bin/cli hash-password mypass` |
| `migrate up` / `migrate down` | Run/rollback DB migrations | `./bin/cli migrate up` |

## Observability & Monitoring

The system includes a pre-configured **Grafana** dashboard and **Prometheus** metrics.

- **Grafana**: [http://localhost:3000](http://localhost:3000)
  - **Credentials**: `admin` / `admin`
  - **Dashboard**: Go to "Dashboards" -> "GoLedger Overview"
- **Prometheus**: [http://localhost:9090](http://localhost:9090)
- **Metrics Endpoint**: [http://localhost:8080/metrics](http://localhost:8080/metrics)

### Dashboard Features
- Real-time Transfer Rate (TPS)
- p95 and p99 Latency
- Active Database Connections
- Total Accounts & Transfers counters

## API Endpoints

Base URL: `http://localhost:8080/api/v1`

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/auth/login` | Log in and receive a JWT |
| GET | `/auth/me` | Get the authenticated user |
| GET | `/ledger/consistency` | Check ledger-wide balance consistency |
| POST | `/accounts` | Create account |
| GET | `/accounts` | List accounts |
| GET | `/accounts/:id` | Get account |
| GET | `/accounts/:id/entries` | List entries for an account |
| GET | `/accounts/:id/transfers` | List transfers for an account |
| GET | `/accounts/:id/balance/history` | Historical balance |
| POST | `/transfers` | Create transfer |
| POST | `/transfers/batch` | Batch transfer (atomic) |
| GET | `/transfers/:id` | Get transfer |
| GET | `/transfers/:id/entries` | List entries for a transfer |
| POST | `/transfers/:id/reverse` | Reverse a transfer |
| POST | `/holds` | Create hold |
| POST | `/holds/:id/capture` | Capture hold |
| POST | `/holds/:id/void` | Void hold |

Unauthenticated: `GET /health`, `GET /ready`, `GET /metrics` (Prometheus).

## Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `DATABASE_URL` | `postgres://ledger:ledger@localhost:5432/ledger?sslmode=disable` | PostgreSQL connection URL |
| `DATABASE_MAX_CONNS` | `25` | Max pool connections |
| `DATABASE_MIN_CONNS` | `5` | Min pool connections |
| `REDIS_URL` | `redis://localhost:6379` | Redis connection URL |
| `HTTP_PORT` | `8080` | HTTP server port |
| `GRPC_PORT` | `50051` | gRPC server port |
| `AUTH_ENABLED` | `false` | Enable JWT authentication |
| `JWT_SECRET` | *(empty)* | JWT signing key — **required** when `AUTH_ENABLED=true`; the server refuses to start otherwise |
| `JWT_EXPIRATION` | `24h` | JWT token lifetime |
| `IDEMPOTENCY_TTL` | `24h` | How long idempotency keys are cached in Redis |
| `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `LOG_FORMAT` | `json` | Log format (json, text) |

Configuration is validated at startup — invalid values (e.g. a non-numeric `HTTP_PORT`, or `DATABASE_MIN_CONNS` greater than `DATABASE_MAX_CONNS`) fail fast with a clear error instead of misbehaving at runtime.

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

## Development

```bash
make deps             # Download dependencies
make generate         # Generate sqlc + mocks
make build            # Build binary
make test             # Run unit tests
make test-integration # Run integration tests (requires Docker)
make test-all         # Run all tests (unit + integration)
make test-coverage    # Generate coverage report (coverage.html)
make coverage-percent # Print total coverage percentage
make lint             # Run golangci-lint
```

## Tech Stack

- **Go 1.24** - Language
- **PostgreSQL 16** - Primary database
- **Redis 7** - Idempotency store
- **chi** - HTTP router
- **sqlc** - Type-safe SQL
- **pgx/v5** - PostgreSQL driver
- **slog** - Structured logging (stdlib)
- **gomock** - Mock generation
- **golangci-lint** - Linting
- **gotestsum** - Test runner

## License

GNU General Public License v3
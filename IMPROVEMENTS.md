# GoLedger — Improvement Analysis

*Analysis date: 2026-07-02, at commit `6803d9e`.*

The project is in good shape overall: clean architecture layering is real (domain → usecase → adapter → infrastructure), the transfer path uses sorted `FOR UPDATE` locking with retry, there's an outbox, audit logging, idempotency middleware, metrics, and both HTTP and gRPC adapters. The items below are ordered by impact.

---

## 1. Correctness bugs (fix first)

### 1.1 Reversal path ignores its own deadlock prevention
`internal/usecase/transfer_usecase.go:490-501` — `executeReverseTransfer` builds and sorts `accountIDs`, then never uses the sorted slice. It locks `fromAccount` and `toAccount` individually, in from→to order:

```go
accountIDs := []string{input.FromAccountID, input.ToAccountID}
sort.Strings(accountIDs)                 // <- result unused
fromAccount, err := uc.accountRepo.GetByIDForUpdate(ctx, tx, input.FromAccountID)
...
toAccount, err := uc.accountRepo.GetByIDForUpdate(ctx, tx, input.ToAccountID)
```

Two concurrent reversals in opposite directions (or a reversal racing a batch transfer) can deadlock. Fix: reuse `GetByIDsForUpdate` with the sorted IDs, as `executeTransferTransaction` does — or better, collapse the reversal into `CreateBatchTransfer` (see §3.1).

### 1.2 Double-reversal race (TOCTOU)
`transfer_usecase.go:428-436` — the `ReversedTransferID != nil` check runs *outside* the transaction, on an unlocked read. Two concurrent `ReverseTransfer` calls for the same transfer both pass the check and both create a reversal, double-crediting the original sender. Migration `000003_add_reversals.up.sql` creates only a plain index, so the DB doesn't stop it either. Fix both layers:

- Add `CREATE UNIQUE INDEX ... ON transfers(reversed_transfer_id) WHERE reversed_transfer_id IS NOT NULL;`
- Re-check (or `SELECT ... FOR UPDATE` the original transfer) inside the transaction.

### 1.3 Caller's metadata map is mutated
`transfer_usecase.go:487` — `metadata["reversal_of"] = originalTransferID` writes into the map the caller passed in. Copy the map before annotating it. Relatedly, in `CreateBatchTransfer` all transfers that fall back to `input.Metadata` share one map reference — harmless today, but fragile.

### 1.4 Balance invariants exist only in Go
`000001_initial.up.sql` has a `CHECK` on transfers (good), but nothing prevents a bug elsewhere from driving `balance` negative on an account with `allow_negative_balance = false`. A DB-level guard is cheap insurance for a ledger, e.g. `CHECK (allow_negative_balance OR balance >= 0)` (and the mirror for `allow_positive_balance`). Same question for `encumbered_balance <= balance` on hold accounting.

---

## 2. Testing & CI

### 2.1 Coverage is 26.8% despite heavy test investment
Recent commits added many adapter/DTO/middleware tests, but total statement coverage is 26.8%. The gap is almost certainly the repository layer and generated-code-adjacent plumbing. Suggestions:

- Focus next on `internal/adapter/repository/postgres/*` using the pgxmock dependency already in `go.mod`, or fold them into the existing `tests/integration` suite (which is decent — concurrent, reversal, outbox, edge cases).
- Set a ratcheting coverage floor in CI (fail below current %, raise as it improves) rather than chasing a fixed number.
- Add a concurrency test specifically for §1.1/§1.2 (parallel reversals of the same transfer) — it would have caught both.

### 2.2 CI Go version matrix is incoherent
`.github/workflows/ci.yml`: lint runs on `1.25.1 / 1.24.4 / 1.23.7` (three redundant lint runs; 1.23 can't even build a `go 1.24.0` module without toolchain auto-download), while the test job runs on `1.22`, below the module's `go 1.24.0` directive. Pick one version (the one in `go.mod`), use it everywhere, and drop the matrix — this is an application, not a library.

### 2.3 Repo hygiene
`coverage.out` is committed to git. Remove it and add `coverage.out` / `coverage.html` / `bin/` to `.gitignore`.

---

## 3. Code quality

### 3.1 ~150 duplicated lines between transfer and reversal
`processTransfer` (transfer_usecase.go:236-364) and `executeReverseTransfer` (471-619) duplicate the entire create-transfer/entries/update-balances/audit sequence, and they've already drifted: the reversal path skips the outbox event (no `transfer.created`/`transfer.reversed` event is emitted for reversals — likely a bug for downstream consumers), and uses `time.Now()` where the main path uses `time.Now().UTC()`. Extract one shared "apply double entry" function, or implement `ReverseTransfer` as a thin wrapper that marks-and-delegates to the batch path. This kills the duplication and fixes §1.1 in one move.

### 3.2 Config validation
`internal/infrastructure/config/config.go` parses env but validates nothing. Notably `AUTH_ENABLED=true` with an empty `JWT_SECRET` should be a startup error, not a runtime surprise. Add a `Validate()` step in `Load()` (secret required when auth enabled, port numeric, conns min ≤ max).

### 3.3 Mixed-currency metrics
`transfer_usecase.go:142-145` observes raw `Amount.Float64()` into one `TransferAmount` histogram. If more than one currency exists, the histogram is meaningless (USD and JPY in the same buckets). Either label by currency or drop the amount histogram and keep count/duration.

### 3.4 Timestamp consistency
The codebase mixes `time.Now()` and `time.Now().UTC()` (e.g. audit logs at transfer_usecase.go:220 vs. entry timestamps at :182). Postgres normalizes `timestamptz`, but in-process comparisons and JSON payloads will differ. Standardize on UTC, ideally via an injectable clock (you already inject `IDGenerator` — a `Clock` interface fits the same pattern and helps tests).

---

## 4. Features / operational maturity (larger, pick as needed)

- **API documentation.** There's no OpenAPI spec for the HTTP API. Since proto definitions exist for gRPC, consider grpc-gateway + generated OpenAPI, or a hand-written `api/openapi.yaml` — right now the README tables are the only contract.
- **Distributed tracing.** Metrics and structured logging are in place; OpenTelemetry tracing (HTTP/gRPC middleware + pgx tracer) is the missing observability leg, and correlation IDs across the outbox would let you follow a transfer end-to-end.
- **Cursor pagination.** `ListTransfersByAccount` uses limit/offset, which degrades on large ledgers and can skip/duplicate rows under concurrent writes. ULIDs are lexically sortable — keyset pagination (`WHERE id < $cursor ORDER BY id DESC`) is a natural fit.
- **Outbox delivery target.** The publisher exists, but check where events actually go and what happens on poison messages — a max-attempts/dead-letter column on the outbox table is standard.
- **Docker Compose v2.** Makefile uses the legacy `docker-compose` binary; migrate targets to `docker compose`.
- **Release/versioning.** `release.yml` exists — consider embedding version/commit via `-ldflags` and exposing it on the health endpoint and CLI `--version`.

---

## Suggested order of attack

1. §1.1 + §1.2 + §3.1 together (one refactor fixes the deadlock, the race, the missing reversal event, and the duplication) with a concurrent-reversal integration test.
2. §1.3, §1.4, §3.2 — small, independent hardening fixes.
3. §2.2 + §2.3 — CI/hygiene, ~30 minutes.
4. §2.1 coverage ratchet, then the §4 items by product priority.

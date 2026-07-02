# GoLedger — Regulatory & Traceability TODO

*Based on architecture review from 2026-07-02 (commit `5d1962b`). See `IMPROVEMENTS.md` for the earlier code-quality analysis.*

**Verdict on architecture:** No event sourcing needed — the append-only `entries` table (with per-entry previous/current balance and account version) already is the event log; `accounts.balance` is a verifiable projection. CQRS only in light form, later, if reporting load ever hurts the OLTP path: project outbox events into a separate reporting store. Do not split command/query stacks preemptively.

---

## P0 — Blockers

- [x] **Wire authentication into the HTTP router**
  `AuthMiddleware` and `RequireRole` are now applied in `internal/adapter/http/router.go`, gated by `AuthEnabled`/`JWTManager` on `RouterConfig` (mirrors the existing `AUTH_ENABLED` config flag; behavior is unchanged when it's off).
- [x] **Wire `AuthInterceptor` into the gRPC server**
  `cmd/server/main.go` now adds `grpcMiddleware.AuthInterceptor` + `MethodRoleInterceptor` to the interceptor chain when `AUTH_ENABLED=true`.
- [x] **Fix audit attribution** (falls out of the two items above)
  Verified via smoke test: with auth enabled, `audit_logs.user_id` is the real authenticated user, not `"system"`.
- [x] **Apply RBAC** — role → endpoint matrix implemented: admin for account creation + `/audit/*`, operator for transfers/holds, viewer for reads. Same matrix mirrored on the gRPC side via `grpcMethodRoles`.

## P1 — Audit-trail hardening

- [x] **DB-enforced append-only** on `entries`, `transfers`, `audit_logs` — migration `000009` adds `BEFORE UPDATE OR DELETE` triggers; verified live (direct SQL UPDATE/DELETE rejected).
- [x] **Audit failed actions** — failure audit rows for transfers/holds/account-create are now written outside the transaction (`auditFailedTransfers`/`auditFailedHold`/`auditFailedAccount`), so a rejected mutation is still recorded.
- [x] **Audit auth events** — login success/failure now audited in `AuthHandler.Login` (also fixed a `context.Background()` bug and a `/auth/me` context-key mismatch that made it always 401).
- [x] **Audit holds and account creation** — already correct; verified `hold_usecase.go`/`account_usecase.go` write audit rows for create/void/capture.
- [x] **Populate `request_id`** — new `domain.RequestMeta` + `middleware.RequestMeta` populate request_id/IP/user-agent into every audit row.

## P2 — Reconciliation (the real "event sourcing" benefit)

- [x] **Replace the `ReconcileAccount` stub** — now computes `balance == SUM(entries.amount)` via `EntryRepository.SumAmountsByAccount`.
- [x] **Group consistency checks by currency** — `CheckLedgerConsistency` now uses `CheckConsistencyByCurrency`, reporting every currency's mismatch instead of one global sum.
- [x] **Entry-chain verification** — `ReconciliationUseCase.VerifyEntryChain` walks each account's entries checking previous/current balance linkage and contiguous `account_version`.
- [x] **Scheduled reconciliation run with alerting** — `internal/infrastructure/reconciliation.Scheduler` runs on `RECONCILIATION_INTERVAL`, alerting via ERROR logs + Prometheus (`goledger_reconciliation_*`).

## P3 — Access for examiners & data lifecycle

- [x] **Audit read API** — admin-only `/api/v1/audit` (+ `/export` CSV, `/resource/:type/:id`, `/user/:userId`) added.
- [x] **Retention policy** — `audit_logs` partitioned by month (migration `000010`), plus `scrub_audit_log_pii()` for GDPR-style minimization and a documented 5-7y compliance retention policy. Row-level UPDATE/DELETE stays blocked; PII scrub uses a narrow, auditable trigger bypass.

## P4 — Nice to have / stronger claims

- [x] **Hash-chain audit rows** — migration `000012`: every row stores `prev_hash`/`hash`/`chain_seq`; `verify_audit_log_chain()` + `./bin/cli audit verify-chain` for tamper detection. Verified live (simulated tamper via bypass, chain break detected).
- [x] **Version outbox events** — `event_version` + per-aggregate `aggregate_sequence` added (migration `000011`); verified sequential (1, 2) across a hold's create→capture lifecycle.
- [x] **OpenTelemetry tracing** — HTTP (`otelhttp`), gRPC (`otelgrpc`), and pgx (`pgx.QueryTracer`) all instrumented, correlated via `trace_id` in the access log. Verified live: HTTP span → nested pgx.query spans, matching trace_id in the log line.
- [x] Remaining `IMPROVEMENTS.md` items: config validation was already done; added cursor pagination (`?cursor=`) on `GET /accounts/:id/transfers`, outbox dead-lettering (`attempts`/`dead_lettered_at`, `./bin/cli outbox dead-letters`), and brought `api/openapi.yaml` up to date with the current routes/schemas.

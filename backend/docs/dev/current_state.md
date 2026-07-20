# PamojaBuild Current State

Last updated: 2026-07-10

## Source Documents Reviewed

- `architecture_v1.md`
- `workflow.md`
- Backend code under `backend/`

## High-Level Status

The backend has a useful scaffold for Phases 1 through 4, plus early placeholder code for later phases. It is not yet production-ready for Lightning payments.

Phase 5, Lightning Integration, is the next real implementation step. The current Lightning code can create fake invoice-looking strings and record fake settlements, but it does not yet talk to an LND node, listen for real invoice settlements, or recover reliably after a restart.

## Phase Review

### Phase 0: Architecture Freeze

Status: Mostly done as documentation.

Evidence:
- `architecture_v1.md` defines the high-level trust model, state machine, Lightning approach, escrow approach, HMAC ledger concept, and frontend API touchpoints.
- `workflow.md` defines the phase order and golden rules.

Gaps:
- Some detailed operational decisions are not fully captured in code yet, especially recovery procedures, exact failure handling, and production deployment practices.

### Phase 1: API Contract First Development

Status: Partially done.

Evidence:
- HTTP handlers and payload structs exist for auth, tasks, volunteer flows, trustees, ledger, Lightning donations, and payout review/signing.
- The donation endpoint exists at `POST /api/v1/tasks/:slug/donate`.

Gaps:
- No OpenAPI/Swagger source-of-truth file was found.
- Response shapes differ from `architecture_v1.md` in places. For example, the donation response currently returns `payment_request`, `payment_hash`, and `expires_at`, while the architecture document names fields like `invoice_bolt11`, `payment_hash_hex`, and includes `task_id`.
- The Lightning status endpoint exists as handler code but is not wired into the router.

### Phase 2: Domain Module Separation

Status: Partially done.

Evidence:
- Separate packages exist for `task`, `trustee`, `ledger`, `lightning`, `escrow`, `auth`, and `volunteer`.
- Most packages have domain interfaces, repositories, services, HTTP handlers, and tests.
- Domain events exist under `backend/internal/events`.

Gaps:
- Some orchestration is currently wired directly in `backend/cmd/app/router.go`.
- Some later-phase behavior is connected too early. For example, financial state changes and signature thresholds can trigger escrow payout placeholder code.
- The architecture says domains should communicate only through events and not directly access each other's database tables. The current code generally tries to follow that, but there are cross-service dependencies and router-level event glue that should be tightened over time.

### Phase 3: Database Before Bitcoin

Status: Partially done.

Evidence:
- PostgreSQL is the only application database and `DATABASE_URL` is required.
- A clean PostgreSQL baseline has paired `golang-migrate` up/down migrations.
- Migrations run through a separate deployment command and no longer run during API startup.
- PostgreSQL integration tests use isolated schemas through `TEST_DATABASE_URL`.
- The schema includes users, tasks, volunteer profiles, applications, submissions, trustee keys, ledger entries, Lightning invoices, volunteer payments, and payout signatures.
- Task creation, volunteer application/submission flow, and ledger HMAC entries are implemented.
- Ledger chain verification exists.

Gaps:
- The task financial state machine is not strictly enforced. `TransitionFinancialState` updates to any target state without validating allowed transitions.
- Ledger balance calculation is basic and may overcount L2 balance after future swaps/payouts because it currently sums entries by type.
- Some repository methods are placeholders, such as ledger balance update and derivation index increment.

### Phase 4: Event Driven Architecture

Status: Partially done.

Evidence:
- An in-memory event bus exists with publish, subscribe, history, and replay.
- Events include task creation, donation/payment settlement, threshold reached, task status changes, and financial state changes.
- Services publish events, and router wiring subscribes to events to record ledger activity.

Gaps:
- The event bus is in-memory only, so events do not survive process restarts.
- Event subscribers are registered in the router instead of a dedicated application/event wiring layer.
- `PaymentSettled` is subscribed twice in `router.go`, which can duplicate ledger entries for a single settlement.
- Event handlers do not consistently handle or log errors from downstream calls.

### Phase 5: Lightning Integration

Status: Foundation work started; real LND integration not implemented yet.

Evidence:
- `backend/internal/lightning` package exists.
- `RequestDonationInvoice` validates positive amounts, saves an invoice record, and returns an invoice-like response.
- `ProcessIncomingSettlement` marks an invoice as settled and publishes `PaymentSettled`.
- Database table `lightning_invoices` exists.
- Invoice records now track status, creation time, expiry time, and LND add/settle indexes.
- Duplicate settlement processing is idempotent at the Lightning service layer.
- `PaymentSettled` now has a single router subscriber.

Gaps:
- No real LND client is implemented.
- Invoice generation is fake.
- Payment hashes are fake.
- No real settlement listener exists.
- Expired invoice cleanup is not implemented yet.
- No restart recovery exists.
- No tests yet for payment timeout or node restart recovery.

## Current Test Baseline

Command attempted:

```sh
cd backend && go test ./...
```

Result: passing as of 2026-07-10.

Notes:
- `backend/go.sum` has been generated.
- Full backend test command completed successfully.
- The latest sandboxed test attempt hit the Go cache permission issue again, then passed when rerun with approved access.

## Phase 5 Starting Point

Before adding real Lightning behavior, the safest next steps are:

1. Keep the clean test baseline green with `cd backend && go test ./...`.
2. Decide the LND connection mode for local development and tests:
   - Real LND on regtest/signet for integration tests.
   - Fake in-memory Lightning client for unit tests.
3. Replace fake invoice generation with a real client interface that can call LND in production and be mocked in tests.
4. Implement an invoice settlement listener that can resume after backend restart.
5. Add expired invoice handling.
6. Add tests required by Phase 5:
   - Payment success.
   - Payment timeout.
   - Node restart recovery.

## Beginner Notes

- A Lightning invoice is like a payment request. The backend creates it, gives it to the donor, and waits for the Lightning node to say it was paid.
- A payment hash is the unique ID for that Lightning invoice.
- Settlement means the invoice was actually paid.
- Idempotent means safe to repeat. If the Lightning node tells us twice that the same invoice was paid, we should credit it only once.
- LND is the Lightning node software this project plans to use.
- The ledger is the project accounting book. Every real movement of sats should create exactly one ledger entry.

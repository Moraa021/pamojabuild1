# Phase 5 Lightning Implementation Plan

Last updated: 2026-07-10

## Goal

Phase 5 makes Lightning donations real.

In simple terms:

1. A donor asks PamojaBuild for a Lightning invoice.
2. The backend asks LND, the Lightning node, to create that invoice.
3. The donor pays the invoice.
4. LND tells the backend the invoice was paid.
5. The backend credits the correct task in the ledger exactly once.

This phase should stay focused on donations coming in. Escrow, on-chain vaults, PSBTs, and payouts belong to later phases.

## Important Words

- Lightning invoice: A payment request that a donor can pay with a Lightning wallet.
- LND: The Lightning node software this backend will talk to.
- Payment hash: The unique ID for a Lightning invoice.
- Settlement: The moment LND confirms the invoice was actually paid.
- Ledger: PamojaBuild's accounting book for task balances.
- Idempotent: Safe to repeat. If the same paid invoice is reported twice, the task should only be credited once.

## Current Starting Point

Already present:

- `backend/internal/lightning` package exists.
- Donation endpoint exists at `POST /api/v1/tasks/:slug/donate`.
- `lightning_invoices` table exists.
- `PaymentSettled` event exists.
- Settled payments can create ledger entries.
- Backend tests currently pass with `go test ./...`.
- Invoice records now include status, creation time, expiry time, and LND index fields for later recovery work.
- Duplicate settlement handling is now safe at the service layer.
- A real LND REST client can create donation invoices.
- Fake Lightning invoice generation is separated into a fake node client for tests.
- Donation invoice creation now goes through the Lightning node client, validates the result, and saves the invoice before returning it.

Still missing:

- Real settlement listener.
- Expired-invoice cleanup and status endpoint behavior.
- Restart recovery.
- Phase 5-specific tests for real payment success, timeout, and restart recovery.

## Implementation Steps

### Step 1: Confirm and Protect the Test Baseline

Status: Done.

`go.sum` has been generated, and `go test ./...` currently passes.

Next work:

- Keep the full backend test suite green before each Phase 5 change.
- Add focused Lightning tests as behavior is introduced.

### Step 2: Fix Duplicate Settlement Wiring

Status: Done.

Problem:

`PaymentSettled` is currently subscribed twice in `backend/cmd/app/router.go`.

Why it matters:

If one Lightning invoice is paid, the backend could write two donation entries to the ledger. That would make the task balance wrong.

Work:

- Removed the duplicate subscription.
- Added service coverage proving duplicate settlement processing does not publish another settlement event.

### Step 3: Improve the Invoice Data Model

Status: Done.

Problem:

The current `lightning_invoices` table is too small for production behavior.

Work:

- Added fields for invoice creation time and expiry time.
- Added LND index fields needed to resume settlement listening after restart.
- Added clear invoice status values: `pending`, `settled`, and `expired`.
- Kept `payment_hash` unique so one invoice cannot be recorded twice.
- Added status/index database indexes for future settlement and recovery queries.

### Step 4: Split Fake Lightning From Real Lightning

Status: Done.

Problem:

The current code creates fake invoice strings directly inside the service/repository layer.

Work:

- Define a clean Lightning node client interface.
- Add a fake client for tests.
- Add a real LND client for production.

Beginner version:

The service should not care whether it is talking to a fake test node or real LND. It should ask for "an invoice for this task and amount" and get back the result.

### Step 5: Generate Real Invoices With LND

Status: Done.

Work:

- Use LND to create real Lightning invoices.
- Include the task slug in invoice metadata so the payment can be connected back to the correct task.
- Save the invoice before returning it to the donor.
- Return the payment request, payment hash, and expiry time from the API.
- Added LND REST client coverage proving the backend sends amount, expiry, macaroon auth, and task metadata to LND.
- Added service validation so malformed node invoice responses are not saved.

Success condition:

A donor can request an invoice, and the response contains a real BOLT11 invoice created by LND.

### Step 6a: Switch Production LND Client To gRPC Before Streaming

Status: Done.

Problem:

Step 5 used LND REST because it was a smaller, dependency-light way to prove real invoice creation through the new `NodeClient` boundary. Step 6 introduces settlement listening, which is naturally a long-running stream of invoice updates. LND's gRPC API is the better production fit for that listener because it is the native LND API, uses typed protobuf messages, and supports streaming cleanly.

Decision:

Before building the paid-invoice listener, evaluate and implement a production `LNDGRPCClient` behind the existing `lightning.NodeClient` interface. The existing REST client can remain as a reference or fallback, but the production listener path should prefer gRPC if dependency and configuration setup are acceptable.

Work:

- Add official LND gRPC/protobuf dependencies.
- Add a `backend/internal/lightning/client/lnd_grpc.go` implementation of `lightning.NodeClient`.
- Reuse the existing service boundary so `LightningService` does not care whether the node client uses REST or gRPC.
- Connect to LND using host, TLS certificate, and macaroon credentials.
- Implement `CreateInvoice` with LND gRPC `AddInvoice`.
- Implement `SubscribeInvoiceSettlements` with LND gRPC invoice subscription streaming.
- Convert LND protobuf invoice responses into the internal `lightning.Invoice` domain type.
- Preserve the existing validation behavior: payment request required, 32-byte payment hash required, task slug stored in our database row.
- Add focused tests using mocks/fakes around the gRPC adapter behavior where possible.
- Decide whether production router wiring should use gRPC by default, while keeping REST available only if explicitly configured.

Result:

- Added the official LND module at `github.com/lightningnetwork/lnd v0.20.2-beta`, which is LND's signed official release naming rather than a Go beta toolchain.
- Mirrored LND's `google.golang.org/protobuf` replacement with `github.com/lightninglabs/protobuf-go-hex-display`, because Go does not inherit `replace` directives from dependency modules.
- Added a gRPC `LNDGRPCClient` that implements the existing `NodeClient` interface.
- Kept the REST client available as `LND_CLIENT_MODE=rest`, but made `LND_CLIENT_MODE=grpc` the default.
- Kept the Lightning service unchanged, because it already depends on the interface rather than a concrete REST or gRPC client.
- Added unit tests for gRPC invoice creation and settlement stream conversion.

Beginner version:

REST was a useful first bridge to LND, but Step 6 needs a live stream of invoice updates. gRPC is better at streams. Because we already created the `NodeClient` interface, switching the real client should not change the Lightning service's business logic.

Success condition:

The backend has a production-ready gRPC LND client that can create invoices and expose a settlement subscription through the existing `NodeClient` interface. After that, Step 6 can wire the background listener without building on the weaker REST streaming path.

### Step 6: Listen for Paid Invoices

Status: Done.

Work:

- Start a background listener when the backend starts.
- Subscribe to invoice updates from LND.
- When LND says an invoice is paid, call the settlement processing code.
- Resume from the latest stored LND settlement index.
- Keep retrying the subscription after recoverable stream errors.
- Stop the listener through the same shutdown context as the HTTP server.

Result:

- `LightningService.StartSettlementListener` now runs the long-lived subscription loop.
- The repository exposes `LatestSettleIndex` so the listener can resume from database state.
- Unknown settled invoices are ignored safely instead of killing the listener.
- `main` now uses `http.Server` with signal-aware graceful shutdown, so the listener and HTTP server share the same lifecycle.
- Router tests still build routers without starting a background listener; the real server path uses `NewRouterWithContext`.

Success condition:

Paying an invoice causes the backend to mark it settled and publish a settlement event.

### Step 7: Make Settlement Idempotent

Problem:

Lightning nodes and backend processes may retry messages. That is normal. The backend must treat repeat messages safely.

Work:

- If an invoice is already settled, return success without writing another ledger entry.
- Make the database update conditional: only change `pending` to `settled`.
- Publish `PaymentSettled` only when this backend instance actually changed the invoice from unpaid to paid.

Success condition:

If the same invoice settlement is processed twice, the ledger still gets exactly one donation entry.

### Step 8: Handle Invoice Timeout

Problem:

Invoices should not stay valid forever.

Work:

- Store `expires_at`.
- Add logic to mark old unpaid invoices as expired.
- Make the invoice status endpoint report expired invoices clearly.

Success condition:

An unpaid invoice after its expiry time is treated as expired, not pending.

### Step 9: Add Restart Recovery

Problem:

The backend may be offline when a donor pays. When it comes back, it must catch up.

Work:

- Store the latest LND settlement position or another reliable recovery marker.
- On startup, ask LND for invoice updates since the last known point.
- Process any paid invoices that were missed while the backend was offline.

Success condition:

If the backend restarts after an invoice was paid, it still records the donation once.

### Step 10: Add Phase 5 Tests

Required tests from `workflow.md`:

- Payment success.
- Payment timeout.
- Duplicate settlement.
- Node restart recovery.

Additional useful tests:

- Invalid donation amount is rejected.
- Unknown payment hash is handled safely.
- Settled invoice publishes exactly one event.
- Settled invoice creates exactly one ledger entry.

## What We Should Avoid During Phase 5

- Do not implement escrow yet.
- Do not build PSBT signing yet.
- Do not trigger payouts from Lightning settlement.
- Do not move real money on mainnet.
- Do not rely on database-only trust for payout approval.

## Done Means

Phase 5 is complete when:

- The backend can create real Lightning invoices through LND.
- The backend records real paid invoices.
- Every settled donation credits the correct task ledger once.
- Duplicate settlement notifications are safe.
- Expired invoices are handled.
- Restart recovery works.
- The Phase 5 test suite passes.

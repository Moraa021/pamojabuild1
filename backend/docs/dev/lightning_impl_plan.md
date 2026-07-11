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

Still missing:

- Real LND client.
- Real BOLT11 invoice generation.
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

Problem:

The current code creates fake invoice strings directly inside the service/repository layer.

Work:

- Define a clean Lightning node client interface.
- Add a fake client for tests.
- Add a real LND client for production.

Beginner version:

The service should not care whether it is talking to a fake test node or real LND. It should ask for "an invoice for this task and amount" and get back the result.

### Step 5: Generate Real Invoices With LND

Work:

- Use LND to create real Lightning invoices.
- Include the task slug in invoice metadata so the payment can be connected back to the correct task.
- Save the invoice before returning it to the donor.
- Return the payment request, payment hash, and expiry time from the API.

Success condition:

A donor can request an invoice, and the response contains a real BOLT11 invoice created by LND.

### Step 6: Listen for Paid Invoices

Work:

- Start a background listener when the backend starts.
- Subscribe to invoice updates from LND.
- When LND says an invoice is paid, call the settlement processing code.

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

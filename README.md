# PamojaBuild

PamojaBuild is a community task and donation platform built around Bitcoin and
the Lightning Network. A creator publishes a local project, volunteers apply
and submit evidence of their work, donors contribute through Lightning
invoices, and task-specific trustees independently verify completion and
authorize payout.

The long-term design combines fast, low-cost Lightning donations with
task-isolated 3-of-5 Bitcoin multisignature escrow. Each task has its own five
trustees: no single trustee, platform account, or unrelated task should be able
to control its on-chain funds.

> [!WARNING]
> PamojaBuild is under active development and is **not ready to hold or move
> real payout funds**. Incoming Lightning invoice handling has a substantial
> implementation, and escrow has working API and persistence scaffolding, but
> secure vault derivation, swaps, PSBT construction, cryptographic signature
> validation, and payout execution are not complete.

## How it works

PamojaBuild coordinates two related but separate lifecycles:

```text
Work lifecycle                         Financial lifecycle

open                                   ACTIVE
  -> in_progress                         -> LIQUIDATING
  -> pending_verification                -> READY_FOR_PAYOUT
  -> completed                           -> PAYOUT_PROCESSING
                                           -> ARCHIVED
```

The work lifecycle records who performs a task and whether the result was
independently verified. The financial lifecycle controls whether donations are
accepted and when funds may proceed toward payout.

The intended end-to-end flow is:

1. A creator publishes a task and nominates five local trustees.
2. Trustees accept their appointments and register valid signing keys.
3. Volunteers apply, are selected, perform the work, and submit evidence.
4. Donors pay task-bound Lightning invoices through a shared LND node.
5. The internal ledger attributes settled sats to the correct task.
6. Larger balances move to a task-specific 3-of-5 on-chain vault; economical
   tail balances may remain on Lightning.
7. An independent trustee verifies the completed work.
8. Trustees review a frozen payout intent, and at least three authorize the
   eventual Layer 1 and Layer 2 payout.

Creator, volunteer, and trustee are relationships scoped to a task rather than
permanent account roles. A creator may also volunteer on their task, but a
trustee may not. Creators and volunteers cannot verify their own work or
authorize their own payout.

## Architecture

```text
Browser (Vite/vanilla JavaScript)
                 |
                 | JSON API + HttpOnly session cookie
                 v
Go API (Gin) -> domain services -> PostgreSQL repositories
                 |                       |
                 |                       +-> tasks, sessions, invoices,
                 |                           ledger and signature records
                 |
                 +-> LND gRPC/REST client -> Bitcoin Lightning Network
```

The backend follows handler → service → repository boundaries. PostgreSQL is
the only application database, and schema changes use paired, versioned
`golang-migrate` migrations. Authentication uses revocable, opaque server-side
sessions delivered in secure HttpOnly cookies.

The planned financial architecture uses one shared LND node for liquidity
efficiency while keeping task ownership separate in the application ledger.
On-chain balances are intended to be cryptographically segregated into
independently derived 3-of-5 vaults.

## Current status

Implemented foundations:

- PostgreSQL schema, versioned migrations, and PostgreSQL integration testing
- General accounts, password authentication, revocable sessions, and
  task-scoped authorization
- Strict snake_case API contracts, safe error envelopes, validation, task
  filters, pagination, and `/auth/me`
- Concurrency-safe work and financial state transitions, immutable transition
  history, readiness checks, and idempotency
- LND gRPC and REST adapters, invoice creation, persisted settlement cursors,
  idempotent settlement processing, and unpaid-invoice expiry
- Frontend pages and components for authentication, campaigns, donations,
  volunteering, trustee actions, payouts, and administration
- Escrow/payout scaffolding that loads task balances and trustee keys, restricts
  signing routes to assigned trustees, persists one signature submission per
  trustee key, counts the 3-of-5 threshold, and publishes a threshold event

Still in progress or planned:

- Trustee nomination, acceptance, key proof, replacement, and rotation
- Volunteer selection, assignment progress, evidence review, and notifications
- Production-safe ledger accounting and reconciliation
- Deterministic xpub derivation and task-specific 3-of-5 vault creation
- Lightning-to-on-chain swaps and UTXO tracking
- Immutable payout intents and real PSBT construction
- Cryptographic validation and combination of trustee signatures
- Layer 1 broadcast, Layer 2 payment, failure recovery, and final accounting
- Refund/cancellation flows, operational hardening, monitoring, and backups

The active implementation sequence is maintained in
[backend/docs/dev/implementation_order.md](backend/docs/dev/implementation_order.md).

## Repository layout

```text
.
├── backend/
│   ├── cmd/                 # API, migration, and seed entry points
│   ├── db/migrations/       # PostgreSQL up/down migrations
│   ├── docs/                # Architecture and development references
│   └── internal/            # Domain packages and infrastructure
└── frontend/
    ├── js/                  # Pages, components, API clients, state, utilities
    ├── styles/              # Shared design system and page styles
    └── tests/               # Unit, integration, and Playwright tests
```

## Local development

### Prerequisites

- Go 1.25 or later
- PostgreSQL
- Node.js 18 or later
- npm
- An LND node and credentials for real Lightning integration

### 1. Configure the environment

Copy the example variables and adjust the database and LND values:

```bash
cp .env.example .env
set -a
source .env
set +a
```

`DATABASE_URL` is required. For local HTTP development,
`SESSION_COOKIE_SECURE=false` must be set explicitly. Do not use the example
`SERVER_SECRET` outside local development.

The application reads process environment variables; it does not load `.env`
itself. If your shell or process manager does not load that file, export the
variables another way.

### 2. Create and migrate PostgreSQL

Create the databases named in `DATABASE_URL` and `TEST_DATABASE_URL`, then run:

```bash
cd backend
make migrate-up
```

The API never runs migrations automatically at startup.

### 3. Run the backend

From `backend/`:

```bash
make run
```

The API listens on `http://localhost:8080` by default. Its health endpoint is
`GET /health`.

### 4. Run the frontend

In another terminal:

```bash
cd frontend
npm install
npm run dev
```

The frontend runs at `http://localhost:3000` and proxies `/api` requests to the
backend during development.

## Testing

Run backend unit tests:

```bash
cd backend
make test
```

Run the PostgreSQL repository and concurrency tests against a dedicated test
database:

```bash
cd backend
export TEST_DATABASE_URL='postgres://user:password@localhost:5432/pamoja_test?sslmode=disable'
make test-postgres
```

Run frontend tests:

```bash
cd frontend
npm test
npm run test:e2e
```

PostgreSQL integration tests are skipped when `TEST_DATABASE_URL` is absent.
CI should always provide it.

## Documentation

- [App flow deep dive](backend/docs/dev/app_flow_deep_dive.md) — the most
  detailed reference for what exists today and what remains
- [Implementation order](backend/docs/dev/implementation_order.md) — current
  progress, upcoming work, and unresolved product decisions
- [Architecture specification](backend/docs/architecture_v1.md) — the intended
  multi-tenant Lightning and escrow design
- [Frontend handoff](backend/docs/dev/frontend_handoff.md) — current API
  contracts and required frontend integration
- [PostgreSQL workflow](backend/docs/dev/postgresql.md) — database setup,
  migrations, and integration tests
- [Lightning testing guide](backend/docs/dev/lightning_testing.md) — manual
  Lightning configuration and verification

When the architecture specification and current code differ, use the app-flow
deep dive and implementation order to determine the implemented state.

## Contributing

Treat financial paths as production-sensitive even during development:

- preserve idempotency in settlement, ledger, and payout flows;
- keep important actions auditable;
- use PostgreSQL-specific integration tests for persistence and concurrency;
- add paired up/down migrations for schema changes;
- update the implementation order and app-flow documentation with each
  completed backend unit; and
- use focused Conventional Commits.

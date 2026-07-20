# PamojaBuild App Flow: Beginner-Friendly Deep Dive

Last reviewed against the code: 2026-07-18

## Why this document exists

This is a map of the application as it exists today, not only a description of the finished product we hope to build.

It explains:

- what a donor, campaign creator, volunteer, and trustee is supposed to do;
- what the frontend sends and receives;
- which backend handler, service, repository, and database table handles the data;
- what Bitcoin, Lightning, xpubs, multisig, PSBTs, HMACs, and the ledger mean;
- which parts work now, which parts are only scaffolding, and what should be built next;
- why the frontend task lifecycle and the database lifecycle currently look different.

The two starting architecture documents are:

- [workflow.md](workflow.md)
- [architecture_v1.md](../architecture_v1.md)

Use those documents for the planned phase order and high-level design. Use this document to understand the current code.

> **Important safety warning:** PamojaBuild is not ready to move real payout funds. Real Lightning invoice ingestion is substantially implemented, but trustee authorization, on-chain escrow, PSBT creation, payout execution, strict state transitions, and several authorization checks are not.

## The 60-second mental model

PamojaBuild connects four main human activities:

1. A **campaign creator** publishes a community task and funding goal.
2. **Donors** pay Lightning invoices. One shared LND node receives the money, while PamojaBuild's database ledger says which task owns which sats.
3. A **volunteer** applies, is selected, does the task, and submits evidence.
4. Five task-specific **trustees** are intended to review the result. At least three must cryptographically approve the final payout.

The intended money can exist in two places:

- **Layer 2 / Lightning:** Fast, cheap payments held in the shared LND node. The database ledger allocates a portion of the shared node balance to each task.
- **Layer 1 / on-chain Bitcoin:** Larger task balances are intended to be moved into a task-specific 3-of-5 multisignature vault.

“3-of-5” means there are five trustees and any three valid trustee keys are enough to spend from the vault. One trustee cannot take the money alone, and two unavailable trustees do not permanently lock it.

The most important separation is:

```text
Work lifecycle                       Money lifecycle
------------------------------       --------------------------------
Who is doing the work?               Where are the task's sats?
Was work submitted/approved?         Can donations or payouts happen?

tasks.status                         tasks.financial_state
task_applications.status             lightning_invoices.status
task_submissions.status              ledger_entries
volunteer_payments.status            payout_signatures
```

These lifecycles should coordinate, but they are not the same thing.

## Current implementation at a glance

| Area | Status today | Plain-language meaning |
|---|---|---|
| Registration and sign-in | Hardened foundation | Accounts are general, phone numbers are canonicalized, passwords are hashed, opaque server-side sessions are revocable per device and delivered in HttpOnly cookies, and `/auth/me` restores display-safe account state. |
| API contracts | Implemented foundation | Registered routes use explicit snake_case DTOs, strict JSON decoding, one safe error envelope, service validation, documented cookie auth, and intentional HTTP status distinctions. |
| Task creation/list/detail | Implemented foundation | Tasks use explicit DTOs and paginated filters. Work and financial state values, versions, legal transitions, history, and idempotent concurrency are enforced. |
| Volunteer application | Partial | A volunteer can apply. No API exists for a creator/admin to approve or reject the application. |
| Work submission | Partial | An approved volunteer can submit evidence. Approval is currently only practical through direct DB changes/tests. |
| Lightning invoice creation | Implemented foundation | Real gRPC and REST LND clients exist, invoices are saved, and status can be queried. |
| Lightning settlement | Implemented foundation | Settlements resume after restart and duplicate notices are handled idempotently. |
| Ledger HMAC chain | Partial security mechanism | Entries are chained and can be verified, but accounting rules and production concurrency/security are incomplete. |
| Trustee assignment | Unsafe scaffold | Five numbered slots can be filled, but there is no real selection/onboarding policy or authorization. |
| xpub derivation / on-chain vault | Not implemented | Xpub strings are stored, but child keys and 3-of-5 addresses are not created. |
| Submarine swaps | Not implemented | No Lightning-to-on-chain transfer is executed or recorded correctly. |
| Payout manifest / PSBT | Placeholder | The API returns literal placeholder values and zero amounts. |
| Trustee signature validation | Not implemented securely | Submitted strings are counted without verifying trustee membership or either signature. |
| Payout broadcast | Placeholder | It prints a log line; it does not combine a PSBT, broadcast Bitcoin, pay Lightning, or update state/ledger. |

## Vocabulary without the jargon

### Bitcoin layers

**Layer 1 (L1)** is the normal Bitcoin blockchain. Transactions are globally recorded and can take time and miner fees. It is useful for strongly isolated, task-specific vaults.

**Layer 2 (L2)** here means the Lightning Network. It uses payment channels above Bitcoin to make payments fast and inexpensive.

**A satoshi (sat)** is the smallest Bitcoin unit. 100,000,000 sats equals 1 bitcoin.

### LND, invoices, and settlement

**LND** is the Lightning node program the backend talks to. Think of it as the payment engine holding Lightning channels and reporting incoming payments.

A **BOLT11 invoice** is a Lightning payment request. It contains information a wallet needs to pay. The long value beginning with something such as `lnbc...` is placed in a QR code.

A **payment hash** is the invoice's unique identifier. PamojaBuild saves:

```text
payment_hash -> task_slug + requested amount
```

That mapping is more important for accounting than the human-readable invoice memo. When LND later reports that hash as paid, the backend knows which task to credit.

**Settlement** means LND confirmed that the invoice was actually paid. Creating or displaying an invoice does not mean money was received.

**Idempotent** means “safe to process more than once.” Networks retry. If LND reports the same settlement twice, PamojaBuild must create only one donation credit.

### Ledger

The **ledger** is PamojaBuild's accounting book. It does not itself hold Bitcoin. It records why the app believes a task owns a number of sats.

Example:

```text
Task: clean-kibera-water

TASK_CREATED          0 sats
INBOUND_DONATION  25,000 sats  reference: payment hash A
INBOUND_DONATION  10,000 sats  reference: payment hash B
```

The real Lightning funds are held by the shared LND node. The ledger says that 35,000 of the shared sats belong to this task.

### HMAC and the chained ledger

An **HMAC** is a secret-key fingerprint of data. The server combines an entry's important fields with a secret and produces bytes that should change if any protected field changes.

Each ledger row also includes the previous row's HMAC:

```text
row 1 HMAC = fingerprint(row 1 data)
row 2 HMAC = fingerprint(row 2 data + row 1 HMAC)
row 3 HMAC = fingerprint(row 3 data + row 2 HMAC)
```

This makes a chain. Editing an old amount breaks that row's HMAC and every later link unless the attacker can recompute the chain with the server secret.

An HMAC is **not encryption**. The ledger values remain readable. It is a tamper-detection seal.

It also does not magically prevent all attacks. If an attacker controls both the database and the application secret, they can rebuild the chain. Production still needs secret management, permissions, backups, monitoring, append-only controls, and independent audit evidence.

### Xpub

An **xpub**, or extended public key, can create many related Bitcoin public keys without exposing the corresponding private keys.

That lets PamojaBuild derive a fresh trustee public child key for vault number 0, 1, 2, and so on:

```text
trustee A xpub -> child public key at index 7
trustee B xpub -> child public key at index 7
...
trustee E xpub -> child public key at index 7
                       |
                       +-> task vault address at index 7
```

An xpub cannot normally sign or spend. The private key stays with the trustee's wallet. However, an xpub is still sensitive privacy data because it can reveal a whole family of addresses and balances.

Today the app only stores xpub text. It does not derive child keys or create vault addresses.

### Multisig

**Multisig** means multiple keys control a Bitcoin output. The planned 3-of-5 vault requires valid signatures from any three of the five task trustees.

The five trustees are intended to be different for each task. That provides **tenant isolation**: trustees for one community task should not control another task's vault.

### PSBT

A **PSBT** is a Partially Signed Bitcoin Transaction. It is a standard container for an unfinished transaction.

Think of it as a transaction package:

1. The backend fills in which coins will be spent, the destination, amount, fees, and vault script.
2. A trustee wallet checks the package and adds a signature.
3. Other trustee wallets add signatures to the same transaction.
4. After at least three valid signatures, the payout engine finalizes it into a broadcastable Bitcoin transaction.

A PSBT is not merely “some hex to approve.” Trustees must be shown and must verify the destination, amount, fee, network, and exact transaction identity. Signatures must be cryptographically validated before being counted.

Today there is no PSBT engine in this repository.

### WebCrypto signature

The browser uses the Web Crypto API to create an ECDSA P-256 key pair. The public key is sent to the backend; the private key is supposed to stay on the trustee's device.

The planned use is to sign a precise Layer 2 payout authorization, so a database boolean alone cannot release the Lightning tail.

Today the browser key is non-exportable but only kept in a JavaScript variable for the current page/session. Navigation loses it. The backend's verification function parses a different public-key format than the frontend exports and then returns `true` without checking the signature. This is not real authorization yet.

### Submarine swap

A **submarine swap** moves value between Lightning and on-chain Bitcoin without treating the two systems as the same balance.

The architecture intends to swap sufficiently large task balances from the shared Lightning node into that task's 3-of-5 on-chain vault. Small remaining amounts (“tail balances”) stay in Lightning because an on-chain fee could consume too much of them.

No swap client, swap state table, vault output tracking, or threshold worker exists today.

## Code architecture: how one request travels

Most backend flows use four layers:

```text
HTTP handler -> service -> repository -> database
                    |
                    +-> event bus -> another service
```

- A **handler** speaks HTTP: it reads JSON, path parameters, and query parameters, then writes an HTTP response.
- A **service** contains business rules such as “amount must be positive.”
- A **repository** contains SQL and maps Go objects to database rows.
- The **event bus** announces that something happened, such as a settled payment.

The main wiring and real registered routes are in [router.go](../../cmd/app/router.go). Domain interfaces and models are under `backend/internal/<domain>/domain.go`. Database structure is under [migrations](../../db/migrations/).

The event bus is in memory and calls subscribers synchronously. Its history disappears when the process stops. It is useful scaffolding, not a durable production message system.

### API contract conventions

Registered API routes never serialize domain or database structs directly.
Handlers map them to explicit response DTOs with snake_case JSON fields. JSON
write requests accept exactly one object and reject unknown fields. This is
security-relevant for actor fields: `creator_id`, `volunteer_id`, trustee
`user_id`, and payout signer public keys cannot be smuggled into otherwise valid
requests and silently ignored.

All API failures use:

```json
{
  "error": "validation_error",
  "message": "request validation failed",
  "fields": {
    "display_name": "is required"
  }
}
```

`fields` is optional. Handlers return safe messages rather than raw SQL,
binding, or infrastructure errors. Status semantics are:

| Status | Meaning |
|---|---|
| `400 Bad Request` | Malformed JSON, unknown fields, invalid query parameters, or service validation failure. |
| `401 Unauthorized` | No valid server-side cookie session; in domain language this means unauthenticated. |
| `403 Forbidden` | Authenticated, but missing the task relationship/capability required for the operation. |
| `404 Not Found` | Route or requested resource does not exist. |
| `409 Conflict` | Duplicate data, incompatible task relationship, non-donatable state, or another current-state conflict. |
| `500 Internal Server Error` | Unexpected repository or infrastructure failure, reported without internal details. |

List endpoints return named arrays. Task listing additionally uses `page`
(default 1) and `page_size` (default 20, maximum 100), with
`total_items`/`total_pages` metadata. Existing `category`, `region`, and
`status` filters apply before pagination.

Swagger comments describe only registered routes and declare `CookieAuth` as
the default `pamojabuild_session` cookie. The deployed cookie name remains
configurable. Browser callers must use credentialed requests; the cookie is
HttpOnly and never appears in response JSON.

### PostgreSQL and migrations

PostgreSQL is the only application database. `DATABASE_URL` is required and must use a `postgres://` or `postgresql://` URL. The API server connects to the existing schema but never applies DDL during startup.

Schema changes use paired, UTC timestamp-versioned `golang-migrate` files. The current clean baseline is [20260718132603_initial_schema.up.sql](../../db/migrations/20260718132603_initial_schema.up.sql), with its rollback in [20260718132603_initial_schema.down.sql](../../db/migrations/20260718132603_initial_schema.down.sql). Operators apply migrations through `go run ./cmd/migrate up`; see [postgresql.md](postgresql.md).

PostgreSQL repository tests create isolated schemas when `TEST_DATABASE_URL` is present. The LND Go dependency still brings an SQLite module transitively for LND's own internal packages, but PamojaBuild neither imports nor selects SQLite.

## Authentication and task relationships

### Current flow

The frontend registration form sends:

```json
{
  "display_name": "Amina",
  "phone_number": "+254700000000",
  "password": "a-long-password"
}
```

Relevant frontend: [registerPage.js](../../../frontend/js/pages/registerPage.js)

The backend deliberately accepts only `phone_number`, `password`, and `display_name`. The auth service:

1. removes harmless visual separators from the phone number and requires international `+` format;
2. hashes the password with bcrypt, a deliberately slow password-hashing algorithm;
3. creates the general `users` account and its optional volunteer profile in one database transaction;
4. creates a random 256-bit session value, stores only its SHA-256 hash, and commits it with the account;
5. sends the original value in a 24-hour `HttpOnly`, `SameSite=Lax` cookie and returns only non-secret account display data.

Relevant backend:

- [auth payloads](../../internal/auth/delivery/http/payloads.go)
- [auth service](../../internal/auth/service/auth_service.go)
- [auth repository](../../internal/auth/repository/postgres.go)
- [auth middleware](../../internal/auth/delivery/http/middleware.go)
- [general accounts migration](../../db/migrations/20260718160340_general_accounts_and_task_role_guards.up.sql)
- [server-side sessions migration](../../db/migrations/20260718165226_create_user_sessions.up.sql)

The cookie contains an opaque value: random text with no user information inside
it. The database stores only a hash in `user_sessions`, so a database reader
cannot directly copy the stored value into a browser. Authentication hashes the
presented cookie and finds one active, unexpired session and its user.

The database stores `phone_number` and `is_admin` on `users`; `is_admin` is a real system-wide capability. Creator, volunteer, and trustee status comes from the `tasks`, `task_applications`, and trustee relationship rows for one specific task.

Sign-out marks only the presented `user_sessions` row revoked and expires the
cookie. Other devices remain signed in. The backend never returns the session
value in JSON, so normal frontend JavaScript cannot read it.

### Remaining mismatches and risks

- The frontend must stop sending or navigating by a registration `role`.
- All application routes except registration/sign-in require a valid session, including task browsing and donating.
- `GET /api/v1/auth/me` now returns `user_id`, `display_name`, and `is_admin` so the frontend can restore display state after refresh. A `401` clears that state.
- Cookie authentication requires exact credentialed CORS origins and CSRF controls. The backend now rejects unlisted origins and browser cross-site mutations; deployment must configure `CORS_ALLOWED_ORIGINS`.
- Production requires HTTPS because session cookies are `Secure` by default. Local HTTP development must explicitly set `SESSION_COOKIE_SECURE=false`.
- The ledger HMAC secret still has an insecure fallback; ledger hardening remains implementation step 8.
- Trustee-only payout routes now check the authenticated account against the task's trustee rows. The incomplete trustee-registration route still allows self-claiming an empty slot; nomination and acceptance are intentionally deferred to trustee onboarding in step 5.
- Endpoint-by-endpoint creator ownership rules will expand as the missing creator management routes are built.
- Session cleanup and a “sign out all devices” operation are not implemented.
- Public task browsing and donating still need an explicit product/API decision.

## Flow 1: Creating a campaign/task

### Frontend

Page: [createCampaignPage.js](../../../frontend/js/pages/createCampaignPage.js)

The creator enters title, description, category, region, optional location, goal sats, maximum volunteers, and volunteer mode.

The frontend sends:

```json
{
  "title": "Repair the community water point",
  "description": "Replace the broken pump...",
  "category": "infrastructure",
  "region": "Kisumu",
  "location_detail": "Near the market",
  "goal_sats": 500000,
  "max_volunteers": 3,
  "volunteer_mode": "approval_required"
}
```

Endpoint: `POST /api/v1/tasks`

### Backend and database

1. Auth middleware validates the server-side cookie session and stores the signed-in account ID in the request context.
2. [task handler](../../internal/task/delivery/http/handler.go) binds the JSON and uses that authenticated ID as `creator_id`; JSON cannot choose a different owner.
3. [task service](../../internal/task/service/task_service.go) turns the title into a URL slug, sets `status = open`, and sets `financial_state = ACTIVE`.
4. [task repository](../../internal/task/repository/postgres.go) inserts the task and its initial work/financial history rows in one transaction.
5. The service publishes `task.created`.
6. A router subscriber records a zero-value `TASK_CREATED` row in `ledger_entries`.

Important `tasks` columns:

| API/Go field | DB column | Use |
|---|---|---|
| authenticated account ID | `creator_id` | Owner of the campaign; derived from the verified session, never request JSON. |
| `title` | `title` | Human-readable task title. |
| generated `slug` | `slug` | Stable URL/business identifier used by most related tables. |
| `status` | `status` | Volunteer/work lifecycle. Starts as `open`. |
| `financial_state` | `financial_state` | Donation/payout lifecycle. Starts as `ACTIVE`. |
| `work_state_version`, `financial_state_version` | same | Monotonic revisions used for conditional state updates and stale-write protection. |
| `goal_sats` | `goal_sats` | Soft fundraising target. It neither starts work nor caps donations. |
| `max_volunteers` | `max_volunteers` | Intended capacity; currently not enforced. |
| `volunteer_mode` | `volunteer_mode` | `open` or `approval_required`; backend application rules currently do not distinguish them. |

Schema: [initial PostgreSQL migration](../../db/migrations/20260718132603_initial_schema.up.sql)

### Response

The handler maps the task domain object into an explicit `TaskResponse`.
Creation and detail responses therefore use stable lowercase snake_case
(`id`, `slug`, `creator_id`, `financial_state`, and so on) rather than Go field
names. Duplicate title-derived slugs return `409 Conflict`.

### Missing work

- Decide who is allowed to create campaigns.
- Create trustee onboarding records/invitations as part of a defined setup workflow.
- Add creator endpoints for managing applications; task progress actions are now registered.

## Flow 2: Browsing and viewing tasks

Endpoints:

- `GET /api/v1/tasks`
- `GET /api/v1/tasks/:slug`

The list endpoint accepts `category`, `region`, and `status` query filters plus
`page` and `page_size`. It returns `{ "tasks": [...], "pagination": {...} }`.
Most frontend pages still fetch the first page and filter in the browser.

The repository reads `tasks`; related donations, applications, trustees, and ledger balances are not joined into the response. Therefore a task detail response does not currently include:

- amount raised;
- volunteer/application counts;
- selected volunteer;
- trustees or filled slots;
- payout readiness;
- donation history.

All task reads are behind session authentication even though the UI labels some pages public/guest-facing.

## Flow 3: Donating with Lightning

### What the donor sees

Page: [donationPage.js](../../../frontend/js/pages/donationPage.js)

The donor enters sats. The frontend sends:

```json
{ "amount_sats": 25000 }
```

Endpoint: `POST /api/v1/tasks/:slug/donate`

The response is:

```json
{
  "payment_request": "lnbc...",
  "payment_hash": "64 hexadecimal characters",
  "expires_at": 1784380000
}
```

`expires_at` is a Unix timestamp in seconds. The frontend QR component displays `payment_request`.

### Backend request path

1. [Lightning handler](../../internal/lightning/delivery/http/handler.go) decodes the explicit request DTO.
2. [Lightning service](../../internal/lightning/service/lightning_service.go) validates `amount_sats > 0`, loads the task, and requires `financial_state = ACTIVE`.
3. Only then does the service ask the configured LND client to create a one-hour invoice.
4. The LND adapter uses gRPC by default or REST if configured.
5. The service validates that LND returned a payment request and a 32-byte payment hash represented by 64 hex characters.
6. [Lightning repository](../../internal/lightning/repository/postgres.go) inserts the invoice into `lightning_invoices`.
7. Only after the DB insert succeeds does the API return the invoice to the donor.

That save-before-return ordering matters: a donor should not receive a payable invoice that the application cannot later connect to a task.

Relevant LND code:

- [gRPC client](../../internal/lightning/client/lnd_grpc.go)
- [REST client](../../internal/lightning/client/lnd_rest.go)
- [fake test client](../../internal/lightning/client/fake.go)

Invoice DB mapping:

| API/LND value | DB column | Why it exists |
|---|---|---|
| BOLT11 string | `payment_request` | What the donor pays. |
| unique hash | `payment_hash` | Primary key and settlement lookup key. |
| requested sats | `amount_sats` | Amount to credit when this known invoice settles. |
| URL slug | `task_slug` | Which task receives the credit. |
| pending/settled/expired | `status` | Invoice lifecycle. |
| boolean | `settled` | Older/redundant settled flag kept alongside status. |
| times | `created_at`, `expires_at`, `settled_at` | Timing and status display. |
| LND sequence numbers | `add_index`, `settle_index` | Recovery and ordered settlement processing. |

Schema: [initial PostgreSQL migration](../../db/migrations/20260718132603_initial_schema.up.sql)

### What happens after the donor pays

The frontend currently does not poll invoice status. `lightningApi.js` exposes invoice creation only, even though the backend route exists:

```http
GET /api/v1/lightning/invoices/status?payment_hash=<hash>
```

In the backend:

1. A background gRPC subscription listens to LND invoice settlement events.
2. It resumes from the last saved settlement index after reconnect/restart.
3. For a known payment hash, the repository conditionally changes `pending` or `expired` to `settled`.
4. If no row changed, it was already handled, so no second event is published.
5. A fresh settlement publishes `payment.settled`.
6. A router subscriber appends `INBOUND_DONATION` to the task's ledger using the payment hash as `reference_id`.
7. `lightning_sync_state` stores the global LND cursor so irrelevant/unknown LND invoices are not replayed forever.

This is one of the stronger current flows: it includes input validation, unique invoice hashes, conditional settlement, retry, expiry, and restart recovery.

### Important gaps

- The backend now treats only `ACTIVE` as donatable. The frontend still considers `LIQUIDATING` donatable and must be corrected.
- No donor identity or donation history is recorded. The invoice belongs to a task but not to a donor.
- The UI does not poll settlement status or show a confirmed donation.
- The invoice response differs from names/examples in `architecture_v1.md`; choose one OpenAPI contract.
- LND credentials and macaroon permissions need production hardening.

## Flow 4: Applying, selecting volunteers, doing work, and submitting evidence

### Applying

The frontend sends:

```json
{ "message": "I have repaired this type of pump before." }
```

Endpoint: `POST /api/v1/tasks/:slug/apply`

The handler derives `volunteer_id` from the authenticated session. The request contains only the application message. A creator can therefore apply to their own task; they do not receive payment merely by being the creator, but must have an explicit volunteer relationship.

If accepted, the backend:

1. checks for an existing application by task and volunteer;
2. inserts `task_applications` with `status = pending`;
3. publishes `application.submitted`;
4. records a zero-amount `APPLICATION_SUBMITTED` ledger entry.

Application columns:

| Column | Meaning |
|---|---|
| `task_slug` | Task being applied to. |
| `volunteer_id` | Authenticated applicant. |
| `message` | Applicant's explanation. |
| `status` | `pending`, intended later `approved` or `rejected`. |
| `applied_at`, `reviewed_at` | Audit timing. |

Schema: [initial PostgreSQL migration](../../db/migrations/20260718132603_initial_schema.up.sql)

### Selecting/approving a volunteer

This is **not implemented as an app flow**.

The repository has a generic `UpdateStatus`, but there is no registered handler/route for a creator to list a task's applicants or approve/reject one. The integration test directly executes SQL to mark an application approved.

`volunteer_mode = open` also does not auto-approve, and `max_volunteers` is not enforced.

The intended implementation should:

- let authorized task creators/admins list applications for their task;
- conditionally approve/reject pending applications;
- enforce max volunteers transactionally;
- define whether `open` mode auto-approves until capacity is full;
- prevent applying to closed/nonexistent tasks;
- publish typed approval/rejection events;
- leave the task `open` until its creator explicitly starts work.

### Submitting work

The frontend sends:

```json
{
  "description": "Replaced the pump seal and tested the water flow.",
  "evidence_urls": ["https://example.com/photo.jpg"]
}
```

Endpoint: `POST /api/v1/tasks/:slug/submissions`

Backend steps:

1. derive `volunteer_id` from the authenticated session;
2. load that user's application for the task;
3. require application status `approved`;
4. insert `task_submissions` with `status = submitted`;
5. publish `submission.created`;
6. append a zero-amount `SUBMISSION_CREATED` ledger entry.

Submitting evidence does not change the task-wide work state. Once every
approved volunteer has at least one submission, the task creator may call
`POST /tasks/:slug/submit-for-verification`.

Schema: [initial PostgreSQL migration](../../db/migrations/20260718132603_initial_schema.up.sql)

### Current submission limitations

- The work lifecycle is now `open -> in_progress -> pending_verification -> completed`.
- `POST /tasks/:slug/start` lets the creator begin work after at least one approved volunteer exists.
- `POST /tasks/:slug/submit-for-verification` lets the creator request review only after every approved volunteer has submitted.
- `POST /tasks/:slug/verify` lets an independent task trustee verify the work; it also closes donations atomically.
- There is no registered route to review/approve/verify a submission.
- The frontend calls `GET /tasks/:slug/submissions`; no such GET route is registered. The backend only offers the authenticated volunteer's full submission list at `GET /volunteers/submissions`.
- The old frontend `POST /tasks/:slug/complete` call must be replaced with the creator's submit-for-verification action.
- The repository method meant to satisfy both application and submission `UpdateStatus` interfaces updates `task_applications`; its separate submission update method has a different name. Future submission review code could update the wrong table.
- Evidence is only URL text. There is no upload storage, content validation, malware handling, access control, or immutable evidence hash.

## The task lifecycle: frontend view versus database reality

### Frontend's displayed journey

The 13-step [VerificationTracker](../../../frontend/js/components/VerificationTracker.js) presents:

```text
registered
-> profile created
-> task found
-> application sent
-> application approved
-> work in progress
-> evidence submitted
-> under verification
-> trustee review
-> co-signing
-> payment broadcast
-> payment received
-> reputation updated
```

This is a user journey, not a single database state machine. It mixes:

- user/profile existence;
- application status;
- task work status;
- presence of a submission;
- financial state;
- inferred payment/reputation results.

The frontend currently hardcodes some tracker facts (`hasApplication: true`, `applicationApproved: true`) on the submission page instead of loading them.

It also treats `LIQUIDATING` as “payment broadcast” after `READY_FOR_PAYOUT`. The architecture defines `LIQUIDATING` before `READY_FOR_PAYOUT`, so that visualization is backwards.

### Current database state fields

Current state is stored on the owning rows, while every task state change is
also appended to `task_state_transitions`:

```text
tasks.status
    open -> in_progress -> pending_verification -> completed

tasks.financial_state
    ACTIVE -> LIQUIDATING -> READY_FOR_PAYOUT
    -> PAYOUT_PROCESSING -> ARCHIVED
    SYSTEM_LOCKDOWN is reserved but has no executable recovery rules

task_state_transitions
    immutable work/financial history with old/new state, version,
    backend-derived actor, reason, retry key, and timestamp

task_applications.status
    pending initially; no app approval route

task_submissions.status
    submitted initially; no app review route

lightning_invoices.status
    pending -> settled
    pending -> expired
    expired -> settled if LND later confirms actual payment

volunteer_payments.status
    table exists; payout flow does not populate it
```

### Enforced work lifecycle

```text
open
  -> in_progress
  -> pending_verification
  -> completed
```

The implemented transition rules are:

- `open -> in_progress`: the creator acts explicitly, with at least one approved volunteer;
- `in_progress -> pending_verification`: the creator acts after every approved volunteer has submitted;
- `pending_verification -> completed`: a task trustee acts, and that account must be neither creator nor volunteer for the task.

Each API action requires an `Idempotency-Key`. The repository locks the task
row, checks the expected state and version, changes current state, and appends
history in the same transaction. Exact retries replay the original result;
stale transitions or key reuse with different data return `409`.

For multiple volunteers, task-wide state alone is not enough. You may need assignments with their own statuses so one volunteer's submission does not change the entire task incorrectly.

### Enforced money-state boundary

From `workflow.md`:

```text
ACTIVE
  -> LIQUIDATING
  -> READY_FOR_PAYOUT
  -> PAYOUT_PROCESSING
  -> ARCHIVED
```

Plain language:

- `ACTIVE`: accept donations; task accounting changes as invoices settle.
- `LIQUIDATING`: stop accepting new donations and reconcile/swap final balances.
- `READY_FOR_PAYOUT`: freeze a specific payout manifest for trustee review.
- `PAYOUT_PROCESSING`: threshold has been reached; execute exactly that approved payout once.
- `ARCHIVED`: payout is confirmed, balances reconcile to zero, and the task is terminal.

The database constrains these values and the task service permits only the
linear predecessor-to-successor transitions. Browser callers cannot select a
financial target state. Independent work verification atomically changes
`pending_verification -> completed` and `ACTIVE -> LIQUIDATING`, which makes
the existing donation service reject new invoices.

The remaining financial transitions are internal boundaries for later
liquidation/payout services and are not yet triggered. The old event subscriber
that invoked payout finalization merely on entering `LIQUIDATING` or
`READY_FOR_PAYOUT` has been removed.

The architecture also mentions `SYSTEM_LOCKDOWN`. It remains a constrained
vocabulary value, but the service rejects attempts to enter it until approved
entry, authorization, and recovery rules exist.

## Flow 5: Who can be a trustee and how assignment works

### Intended trust model

Each task should have five real, task-specific community trustees. “Who can be one?” is currently a product/security policy question that the code does not answer.

A safe onboarding policy should define:

- who nominates trustees;
- how identity/community membership is checked;
- whether a trustee can also be the creator or volunteer;
- conflict-of-interest rules;
- trustee acceptance and informed consent;
- slot replacement, recovery, removal, and key rotation;
- what happens if fewer than three trustees remain available;
- whether the same person can hold multiple slots (normally no);
- how users verify they are attaching keys to the correct task and network.

### Current frontend and API

Page: [trusteeDashboardPage.js](../../../frontend/js/pages/trusteeDashboardPage.js)

An authenticated user selects slot `0` through `4`, pastes an xpub, generates a browser key pair, and sends:

```json
{
  "trustee_index": 2,
  "xpub": "xpub...",
  "web_crypto_pubkey_hex": "04..."
}
```

Endpoint: `POST /api/v1/tasks/:slug/trustees`

Backend:

1. checks the index is 0–4;
2. checks whether that task/index exists;
3. inserts into `trustee_keys`;
4. publishes `trustee.registered`.

The table's primary key `(task_slug, trustee_index)` ensures only one row per slot.

Schema: [initial PostgreSQL migration](../../db/migrations/20260718132603_initial_schema.up.sql)

The backend now ignores any browser-supplied identity and uses the authenticated account ID. The database also rejects either direction of a same-task trustee/volunteer conflict.

The registration response is an explicit object containing only `task_slug`,
`trustee_index`, and the authenticated `user_id`; key material is not echoed.

### Why it is still unsafe today

- There is no invitation/nomination record proving the user was selected.
- Any authenticated account can still self-claim an empty slot until step 5 replaces this scaffold with nomination and acceptance.
- The schema prevents the same user from occupying multiple slots for one task.
- Xpub is only checked superficially in the browser; backend does not parse it, validate network/version, or prove key ownership.
- Browser public keys are not validated at registration.
- Slot assignment checks then writes in separate operations, creating a race. The repository also uses an upsert, so direct/concurrent behavior can replace a slot.
- There is no trustee list route registered; the earlier unregistered handler was removed so generated API documentation does not advertise a nonexistent endpoint.
- No replacement/key rotation history or revocation exists.
- Private browser keys are not persisted safely. The generated key is stored in a local variable and is not connected to the payout page's separate `_sessionPrivateKey` variable.

### Intended implementation

Use an explicit task trustee assignment record with states such as:

```text
INVITED -> ACCEPTED -> KEYS_REGISTERED -> ACTIVE
                               |
                               +-> REVOKED / REPLACED
```

Bind it to authenticated user ID, enforce one user per task, validate both public keys, require proof of possession, store key version/history, and make slot claiming a single conditional transaction.

## Flow 6: Ledger accounting and security

### Entries created today

Router event subscribers create:

| Event | Ledger `entry_type` | Amount |
|---|---|---|
| Task created | `TASK_CREATED` | 0 |
| Application submitted | `APPLICATION_SUBMITTED` | 0 |
| Work submitted | `SUBMISSION_CREATED` | 0 |
| Lightning payment settled | `INBOUND_DONATION` | invoice sats |
| Task status changed to completed | `TASK_STATUS_COMPLETED` | 0 |

The schema is in the [initial PostgreSQL migration](../../db/migrations/20260718132603_initial_schema.up.sql).

The HMAC input in actual code is:

```text
task_slug : entry_type : amount_sats : reference_id + previous row HMAC
```

Unlike the formula in `architecture_v1.md`, the actual HMAC does not include a timestamp, and the table has no ledger timestamp column.

### Balance calculation today

[ledger repository](../../internal/ledger/repository/postgres.go) calculates:

```text
L2 balance = sum of all INBOUND_DONATION amounts
L1 balance = sum of all SUBMARINE_SWAP amounts
current index = maximum ledger row ID
```

This is not sufficient accounting:

- a swap should reduce L2 and increase L1, but current sums do not reduce L2;
- payouts and fees are not subtracted;
- reversals/corrections are undefined;
- “current index” is a global ledger row ID, not an address derivation index;
- `UpdateBalances` and `IncrementDerivationIndex` are no-op placeholders;
- entry types and amount signs are unconstrained strings/integers.

### Current integrity protection

`GET /api/v1/ledger/tasks/:slug/verify` recalculates the task's chain and compares HMACs using constant-time comparison.

Good current ideas:

- every settled known invoice is conditionally handled once;
- payment hash is used as an external reference;
- entries are ordered and chained;
- verification is available before future money movement;
- the public API does not expose an endpoint for arbitrary ledger writes (the handler exists but is not registered).

Important limitations:

- a Go process-local mutex prevents concurrent appends only inside one server process; multiple instances can race and create ledger forks;
- append and “read previous row” are not one serializable database transaction;
- no unique constraint prevents two `INBOUND_DONATION` entries for the same payment hash;
- event subscribers ignore ledger write errors, so an invoice could become settled while its ledger credit fails;
- events are in memory, so there is no durable retry/outbox to repair that failure;
- the default HMAC secret is public in source defaults;
- verification is manual and payout placeholder code does not call the ledger security service;
- rows can be deleted from the end without breaking the remaining chain unless an independent checkpoint says what the last row should be;
- no timestamps, actor, metadata, sequence-per-task, or explicit debit/credit accounts exist.

### Intended secure ledger design

Before real money, use a database transaction that:

1. locks the task ledger/account row;
2. checks an idempotency key such as `lightning:<payment_hash>`;
3. reads the last task sequence/HMAC;
4. validates the permitted accounting operation;
5. inserts an immutable entry with actor, timestamp, sequence, and metadata;
6. updates or derives account balances consistently;
7. writes a durable outbox event in the same transaction;
8. commits once.

Use unique constraints for external references, production secret management, periodic independently stored chain checkpoints, least-privilege DB permissions, monitoring, and reconciliation against LND/on-chain truth.

The ledger should make tampering detectable and accounting reproducible. It should not be the only barrier that authorizes a payout.

## Flow 7: Moving Lightning value into an on-chain vault

This is intended Phase 6 work and is not implemented.

The desired sequence is:

1. Five active trustee xpubs exist for the task.
2. The escrow service reserves a task-specific derivation index.
3. It derives one child public key from each xpub at that exact index.
4. It builds the 3-of-5 Bitcoin locking script and address for the configured network.
5. A swap provider/client moves task-attributed Lightning value to that address.
6. The backend tracks swap states and confirms the on-chain output.
7. Ledger entries atomically reduce L2 and increase L1, including all fees.
8. The vault script, derivation path, transaction ID, output index, and amount are stored for later PSBT construction/recovery.

The current `AddressDerivationService` is only an unused interface. There is no implementation or database model for vaults, UTXOs, swaps, or derivation reservations.

## Flow 8: Preparing, approving, and executing payout

### Intended review manifest

When work is verified and finances are reconciled, the backend should create an immutable payout intent containing:

- task and payout ID;
- exact volunteer/beneficiary;
- exact on-chain destination and amount;
- exact Lightning invoice/payment hash and amount;
- task vault UTXOs;
- miner/routing fees and policies;
- unsigned PSBT;
- expiry/version/nonce;
- a canonical digest trustees sign.

The API should return the same stored intent to every trustee. Query parameters supplied by a trustee must not silently create a different destination.

### Current manifest endpoint

Endpoint:

```http
GET /api/v1/trustees/payouts/:slug
```

The handler no longer accepts `destination_address` or `volunteer_invoice`
query values. A later payout-intent workflow must derive and freeze those
server-side rather than allowing a review URL to select destinations.

It requires five trustee rows and reads the basic ledger balance. It then returns
an explicit placeholder DTO:

```json
{
  "task_slug": "some-task",
  "unsigned_psbt_hex": "unsigned_psbt_placeholder",
  "volunteer_invoice": "volunteer_invoice_placeholder",
  "l1_amount_sats": 0,
  "l2_amount_sats": 0
}
```

This is UI scaffolding only.

### Current approval submission

Endpoint:

```http
POST /api/v1/trustees/payouts/:slug/sign
```

Request:

```json
{
  "layer1_psbt_signature_fragment": "some text",
  "layer2_web_crypto_signature": "some text"
}
```

The backend derives the signer key from the authenticated account's trustee row
for the task and stores the placeholder fragments in `payout_signatures`. The
primary key is `(task_slug, trustee_public_key_hex)`, so repeating a submission
from that stored trustee key updates one row.

Then it counts rows. At three rows it publishes `threshold.reached`, whose router subscriber calls payout finalization.

### Why this is not approval yet

The backend now binds a submitted row to the authenticated task trustee's stored
key, but it still does **not**:

- prove that the caller owns the stored key;
- verify the Layer 2 signature;
- parse or verify the PSBT fragment;
- ensure all signatures cover the same payout intent;
- check task state or ledger HMAC integrity;
- make threshold publication one-time;
- store a payout ID/version, timestamps, status, or signed digest.

The existing trustee `VerifyWebCryptoSignature` function is not used here and returns `true` after parsing a key without verifying the signature. Frontend raw P-256 public keys and backend X.509 parsing also disagree.

### Current finalization

`FinalizeAndBroadcastPayout` loads rows, requires at least three, prints:

```text
Broadcasting payout for <task> with <count> signatures
```

and returns success.

It does not move funds or update any tables.

### Intended secure payout sequence

```text
verified work
-> LIQUIDATING: stop donations and reconcile
-> READY_FOR_PAYOUT: persist one frozen payout intent + PSBT
-> trustees independently review the human-readable intent
-> validate each L1 and L2 signature against that intent and assigned trustee
-> 3 unique valid trustees reached
-> atomically claim payout execution and enter PAYOUT_PROCESSING
-> re-verify ledger, balances, intent, signatures, and idempotency
-> finalize/broadcast L1 and pay L2
-> persist txid/payment hash and ledger debits
-> reconcile external confirmations
-> create volunteer payment/update reputation
-> ARCHIVED
```

Failures need explicit resumable states. If L1 broadcasts but L2 fails, retry only the missing L2 leg; never recreate or double-send the L1 transaction.

## Frontend-to-database mapping cheat sheet

| User action | Frontend source | API | Backend path | Main tables |
|---|---|---|---|---|
| Register/sign in/restore | auth pages | `/auth/register`, `/auth/signin`, `/auth/me` | auth handler/service/repo | `users`, `user_sessions` |
| Create campaign | create campaign page | `POST /tasks` | task handler/service/repo | `tasks`, then `ledger_entries` |
| Browse/detail | task pages/stores | `GET /tasks*` | task handler/service/repo | `tasks` |
| Apply | volunteer detail modal | `POST /tasks/:slug/apply` | volunteer application service/repo | `task_applications`, `ledger_entries` |
| Approve application | No working UI/API | Not registered | Repository capability only | `task_applications` |
| Start work | Frontend integration required | `POST /tasks/:slug/start` | task state service/repo | `tasks`, `task_state_transitions` |
| Submit work | submission page | `POST /tasks/:slug/submissions` | submission service/repo | `task_submissions`, `ledger_entries` |
| Request verification | Frontend integration required | `POST /tasks/:slug/submit-for-verification` | task state service/repo | `tasks`, `task_state_transitions` |
| Verify work | Frontend integration required | `POST /tasks/:slug/verify` | task state service/repo | `tasks`, `task_state_transitions` |
| View state history | Frontend integration required | `GET /tasks/:slug/state-history` | task state service/repo | `task_state_transitions` |
| Create donation invoice | donation page/store | `POST /tasks/:slug/donate` | Lightning service/LND/repo | `lightning_invoices` |
| Confirm donation | UI missing polling | `GET /lightning/invoices/status` exists | LND listener + event subscriber | `lightning_invoices`, `lightning_sync_state`, `ledger_entries` |
| Register trustee keys | trustee page/store | `POST /tasks/:slug/trustees` | trustee service/repo | `trustee_keys` |
| Generate vault/swap | No UI/API | None | Not implemented | Tables missing |
| Review payout | payout page/store | `GET /trustees/payouts/:slug` | escrow placeholder | reads `trustee_keys`, `ledger_entries`; returns placeholders |
| Sign payout | payout page/store | `POST /trustees/payouts/:slug/sign` | escrow placeholder | `payout_signatures` |
| Execute payout | Event-triggered placeholder | None | escrow log only | no meaningful writes |
| View earnings | volunteer pages | `GET /volunteers/payments` | explicit payment list | `volunteer_payments` (normally empty until payout work) |

## Database table guide

| Table | Purpose today | Key gaps |
|---|---|---|
| `users` | General login identity, password hash, display name, and global admin flag | Formal admin promotion/removal workflow is not implemented. |
| `user_sessions` | SHA-256 hashes of active/revoked per-device login sessions and their expiries | Cleanup, device labels, “sign out all,” and session-management UI are not implemented. |
| `tasks` | Campaign details plus constrained work/financial current states and monotonic versions | Payout linkage and emergency recovery policy remain missing. |
| `task_state_transitions` | Immutable, paginated state audit history and idempotency records | Actor identity for later internal financial automation may be null by design; durable event delivery remains separate work. |
| `volunteer_profiles` | Bio, skills, payout addresses, reputation totals | Auto-created with the account; partial updates overwrite unrelated values. |
| `task_applications` | Volunteer requests to join tasks | Unique per task/account and conflict-guarded against trustees; no approval API; capacity rules missing. |
| `task_submissions` | Work descriptions and evidence URL arrays | No review API; weak evidence model; multiple-submission policy unclear. |
| `trustee_keys` | Five indexed user/xpub/browser-key rows per task | Unique user per task and conflict-guarded against volunteers; no invitation, key uniqueness, rotation, validation, or history. |
| `ledger_entries` | HMAC-chained task event/accounting rows | Accounting model, concurrency, unique references, timestamps, and durable recovery incomplete. |
| `lightning_invoices` | Incoming Lightning requests and settlement state | Donor identity/history and reconciliation UI missing. |
| `lightning_sync_state` | Durable LND settlement cursor | Appropriate foundation; needs operational monitoring. |
| `volunteer_payments` | Intended outgoing payment records | Not integrated into payout or returned as history. |
| `payout_signatures` | Unverified strings submitted by claimed public key | Missing payout identity, validation, signer binding, timestamps, and status. |

Tables still needed or needing redesign for later phases likely include:

- task state transition history;
- trustee invitations/assignments and key versions;
- task vaults, scripts, derivation paths, and UTXOs;
- swap attempts and confirmations;
- payout intents/manifests and payout execution attempts;
- validated signatures linked to payout version and trustee assignment;
- durable outbox/inbox/idempotency records;
- richer audit logs/checkpoints.

## Security controls: present versus still required

### Present foundations

- bcrypt password hashing;
- random, hashed, revocable 24-hour server-side sessions;
- `HttpOnly`, `Secure`-by-default, `SameSite=Lax` cookies;
- exact-origin credentialed CORS and browser cross-site mutation rejection;
- authenticated API group;
- request validation tags and positive donation amounts;
- a global rate-limiter middleware hook exists, but its current implementation is a no-op and provides no real rate limiting;
- LND TLS and macaroon configuration;
- unique Lightning payment hash;
- conditional/idempotent settlement handling;
- authorized, conditional, idempotent task transitions with row locking, version checks, and immutable history;
- restart settlement cursor;
- HMAC-chained ledger and constant-time comparison;
- task-scoped trustee slot primary key;
- browser-generated non-exportable private WebCrypto keys.

### Not yet sufficient

- endpoint role and ownership authorization;
- production secret enforcement/rotation;
- correct signature verification;
- PSBT parsing and signature validation;
- xpub/network/ownership validation;
- approved `SYSTEM_LOCKDOWN` entry and recovery rules;
- durable event delivery and atomic ledger integration;
- multi-instance-safe ledger append;
- reliable audit logs and independent checkpoints;
- payout idempotency and partial-failure recovery;
- address/invoice validation against Bitcoin network and exact amount;
- deployment validation for allowed frontend origins and HTTPS;
- least-privilege LND macaroon and DB accounts;
- encrypted/sensitive xpub handling policy;
- monitoring, backups, reconciliation, incident lockdown, and key recovery;
- security review before any real-fund deployment.

## High-priority inconsistencies to fix before building later phases

### Priority 0: make the current API/task flow coherent

1. Complete frontend integration with general accounts, credentialed cookie sessions, and `/auth/me`.
2. Implement creator application review; the application request/response contract is now standardized.
3. Integrate the frontend with the enforced work lifecycle and its idempotency headers.
4. Align frontend submission/history/complete displays with the registered state endpoints.
5. Make task browsing/donation publicity an explicit product decision.

### Priority 1: finish safe incoming-money accounting

1. Add frontend invoice status polling/confirmation.
2. Make settlement-to-ledger atomic or durably retryable with an outbox/inbox.
3. Add unique ledger idempotency references and multi-instance-safe appends.
4. Redesign balance accounting around explicit debits/credits and fees.
5. Reconcile the database against LND and alert on drift.

### Priority 2: freeze the trustee and escrow design before coding payout

1. Decide trustee eligibility, nomination, conflicts, replacement, and recovery.
2. Implement authenticated invitation/acceptance and proof of key ownership.
3. Design vault, derivation, UTXO, and swap tables.
4. Build/test deterministic 3-of-5 address derivation on regtest/signet.
5. Implement swaps separately from payout.

### Priority 3: build the PSBT engine and payout orchestrator

1. Persist versioned, immutable payout intents.
2. Construct standards-compliant PSBTs from known vault UTXOs.
3. Define one canonical authorization digest for both payout legs.
4. Validate assigned trustee identity plus L1/L2 signatures.
5. Count three distinct valid trustees only.
6. Implement a single idempotent executor with partial-failure recovery.
7. Record payout debits, confirmations, volunteer payment, state, and audit evidence.
8. Test invalid, duplicate, missing, replayed, and cross-task signatures.

## Suggested source reading order

If time is short, read in this order:

1. [router.go](../../cmd/app/router.go) — every real route and cross-domain event connection.
2. [initial PostgreSQL migration](../../db/migrations/20260718132603_initial_schema.up.sql) and [task domain](../../internal/task/domain.go) — the two task state fields.
3. [Lightning service](../../internal/lightning/service/lightning_service.go) and [repository](../../internal/lightning/repository/postgres.go) — the best-developed money flow.
4. [ledger service](../../internal/ledger/service/ledger_service.go) and [repository](../../internal/ledger/repository/postgres.go) — HMAC chain and current balances.
5. [trustee service](../../internal/trustee/service/trustee_service.go) and [trustee page](../../../frontend/js/pages/trusteeDashboardPage.js) — current assignment/key gaps.
6. [escrow service](../../internal/escrow/service/escrow_service.go), [handler](../../internal/escrow/delivery/http/handler.go), and [payout page](../../../frontend/js/pages/payoutReviewPage.js) — clearly see the payout placeholders.
7. [volunteer services](../../internal/volunteer/service/) and [volunteer repository](../../internal/volunteer/repository/postgres.go) — applications and submissions.
8. [VerificationTracker.js](../../../frontend/js/components/VerificationTracker.js) — the frontend-only combined journey.

## Final takeaway

PamojaBuild currently has a promising modular scaffold and a meaningful Lightning ingestion foundation. The code can create and track Lightning invoices, recover settlement listening after restart, and credit an HMAC-chained task ledger once per normal settlement path.

The rest of the user journey is much less complete. Volunteer approval, trustee
selection/onboarding, on-chain vault creation, swaps, PSBTs, cryptographic
payout approval, payout execution, and final reconciliation are not finished.
Some screens make these features look more complete than they are because they
are wired to placeholder responses or nonexistent endpoints.

The next dependency is trustworthy trustee nomination and acceptance; the
state machine currently recognizes the existing task-trustee relationship, but
that relationship is still created through an unsafe self-claim scaffold.
After trustee and volunteer workflows, harden incoming accounting, freeze the
vault/payout data model, build escrow and PSBT engines, and only then connect
them through one idempotent payout orchestrator.

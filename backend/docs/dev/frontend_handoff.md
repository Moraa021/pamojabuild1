# Frontend Handoff

This file records frontend work required by completed backend implementation units.

## 1. PostgreSQL and migrations

No frontend API, payload, response, route, or UI changes are required.

Deployment environments must run the backend migration command before starting a backend version that expects a new schema. This is an infrastructure/backend responsibility, not a browser integration.

## 2. Accounts and authorization

Registration creates a general account. Remove the `role` selector/value from registration and do not treat a user as a permanent creator, volunteer, or trustee. Those relationships are determined separately for each task.

Authentication responses are now:

```json
{
  "user_id": 12,
  "is_admin": false,
  "display_name": "Amina",
  "expires_at": "2026-07-19T16:03:40Z"
}
```

Authentication is held in an `HttpOnly` session cookie. JavaScript must not read,
store, or send an authentication token.

Store `is_admin` only for genuine system-administration UI. Do not use it to display trustee or volunteer dashboards. Those dashboards will use task relationship data added by later workflow endpoints.

Request changes:

- `POST /api/v1/tasks`: remove `creator_id`; the backend uses the signed-in account.
- `POST /api/v1/tasks/:slug/apply`: send only `message`; remove `volunteer_id`.
- `POST /api/v1/tasks/:slug/trustees`: remove `user_id`; the backend uses the signed-in account. This route remains an incomplete scaffold until trustee onboarding is implemented.
- All API requests, including registration and sign-in, must use Fetch
  `credentials: "include"` (or Axios `withCredentials: true`). This allows the
  browser to accept and send the session cookie.
- Remove the JWT token accessor, `Authorization: Bearer ...` injection, and
  token storage in `sessionStorage`.
- `POST /api/v1/auth/signout`: send no JSON body. A successful response is
  `204 No Content`; the backend revokes the current session and expires its
  cookie.
- Frontend state may retain non-sensitive display data such as `user_id`,
  `display_name`, and `is_admin`, but it is not proof of authentication.
- On initial page load, the frontend will eventually need a `GET /auth/me`
  endpoint to restore display state from the cookie. That endpoint is now
  implemented in step 3 as described below.

Phone numbers must use international format beginning with `+`. Spaces, parentheses, and hyphens are accepted and normalized by the backend.

Trustee payout review/signing now returns `403 Forbidden` unless the signed-in account is a trustee for that task. The frontend should show a simple “You are not assigned as a trustee for this task” message and must not treat admin status as trustee approval.

Deployment integration:

- The frontend origin must be listed exactly in backend
  `CORS_ALLOWED_ORIGINS`, as a comma-separated list.
- Production cookies are `Secure` by default and therefore require HTTPS.
- Local HTTP development must run the backend with
  `SESSION_COOKIE_SECURE=false`.
- The frontend and API should remain on the same site (for example
  `app.example.org` and `api.example.org`) so the `SameSite=Lax` cookie works
  without weakening it to a cross-site cookie.

## 3. API contracts

Every API error now uses one safe envelope:

```json
{
  "error": "validation_error",
  "message": "request validation failed",
  "fields": {
    "display_name": "is required"
  }
}
```

`fields` is present only for field-specific validation failures. Stable `error`
values include `validation_error`, `unauthenticated`, `unauthorized`,
`not_found`, `conflict`, and `internal_error`. Do not display a different
message based only on a caught JavaScript exception type. Use the HTTP status
and `error` value:

- `400`: malformed input, unknown JSON fields, invalid filters, or service validation.
- `401`: no valid cookie session; clear local display state and navigate to sign-in.
- `403`: signed in, but not allowed to perform this task-specific operation.
- `404`: the requested route, task, invoice, or profile does not exist.
- `409`: the request conflicts with current data or task state.
- `500`: unexpected server failure; show a generic retry message.

The API rejects unknown JSON fields. This deliberately prevents old identity
fields such as `creator_id`, `volunteer_id`, `user_id`, and
`trustee_public_key_hex` from being silently ignored. Remove those fields at
their callers.

### Cookie session restore

Add this call:

```http
GET /api/v1/auth/me
```

Response:

```json
{
  "user_id": 12,
  "is_admin": false,
  "display_name": "Amina"
}
```

Call it once during application startup with credentials enabled. A `200`
restores display-only account state. A `401` means the browser has no current
session and should clear that state. Do not expect or store a token, phone
number, password hash, or permanent creator/volunteer/trustee role.

All requests must set Fetch `credentials: "include"` or Axios
`withCredentials: true`, including registration and sign-in so the browser can
accept `Set-Cookie`. Remove `Authorization: Bearer`, the token accessor,
`sessionStorage` authentication, and the legacy role-based auth store.

`POST /api/v1/auth/signout` has no request body and returns `204 No Content`.

### Tasks and pagination

Task create/detail responses are explicit snake_case objects:

```json
{
  "id": 7,
  "slug": "repair-community-pump",
  "creator_id": 12,
  "title": "Repair community pump",
  "description": "Replace the broken seal",
  "category": "infrastructure",
  "region": "Kisumu",
  "location_detail": "Near the market",
  "status": "open",
  "financial_state": "ACTIVE",
  "work_state_version": 1,
  "financial_state_version": 1,
  "goal_sats": 500000,
  "max_volunteers": 3,
  "volunteer_mode": "approval_required",
  "created_at": "2026-07-18T17:30:00Z"
}
```

`GET /api/v1/tasks` accepts `category`, `region`, `status`, `page` (default 1),
and `page_size` (default 20, maximum 100). Its response is:

```json
{
  "tasks": [],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total_items": 0,
    "total_pages": 0
  }
}
```

Both campaign and volunteer task stores must unwrap `tasks`. Use the returned
pagination object when adding server-driven paging. `POST /tasks` must not send
`creator_id`. A duplicate title-derived slug returns `409`.

### Volunteer relationships

`POST /api/v1/tasks/:slug/apply` accepts only:

```json
{ "message": "I can help repair this pump." }
```

`GET /api/v1/volunteers/applications` returns:

```json
{ "applications": [] }
```

The application store currently treats the whole response as an array; unwrap
`applications`.

`POST /api/v1/tasks/:slug/submissions` continues to accept `description` and
one to twenty HTTP(S) `evidence_urls`. A signed-in account without an approved
application now receives `403`, not `400`.

`GET /api/v1/volunteers/submissions` returns:

```json
{ "submissions": [] }
```

Use that registered route and unwrap `submissions`. The current frontend call
to `GET /tasks/:slug/submissions` is not implemented. Filter the authenticated
account's returned list by `task_slug` if the current page needs one task. The
current `POST /tasks/:slug/complete` caller still targets no backend route;
replace it with the state actions described in step 4 below.

`GET /api/v1/volunteers/payments` now returns actual payment records:

```json
{ "payments": [] }
```

Unwrap `payments`. It will normally be empty until the later payout workflow
creates payment records; do not infer completed payments from the profile
summary.

`GET /api/v1/volunteers/profile` and `PUT /api/v1/volunteers/profile` both
return the explicit profile object. `skills` is always an array.

Payment profile routes are:

```http
GET /api/v1/volunteers/payment-profile
PUT /api/v1/volunteers/payment-profile
```

The `PUT` body accepts `lightning_address` and `onchain_address` and returns
those saved values. Use `PUT`, not the current frontend `POST`. Remove
`preferred_method`; the backend does not store it and unknown fields are
rejected. At least one address must be non-empty. Saving payment addresses no
longer overwrites bio or skills.

### Donations

`POST /api/v1/tasks/:slug/donate` still returns
`payment_request`, `payment_hash`, and Unix-seconds `expires_at`. The backend
now checks task existence and requires `financial_state = "ACTIVE"` before
calling LND. Remove `LIQUIDATING` from the frontend's donatable states. A
missing task returns `404`; a non-active task returns `409`.

Add polling through:

```http
GET /api/v1/lightning/invoices/status?payment_hash=<64-hex-character-hash>
```

The response contains `payment_hash`, `status`, `settled`, and optional
Unix-seconds `expires_at`/`settled_at`. Stop polling on `settled` or `expired`,
and show payment confirmation only after `settled`.

### Trustee and payout scaffolds

`POST /api/v1/tasks/:slug/trustees` still accepts `trustee_index`, `xpub`, and
`web_crypto_pubkey_hex`, and now returns:

```json
{
  "task_slug": "repair-community-pump",
  "trustee_index": 2,
  "user_id": 12
}
```

This route was removed by step 5. Do not keep using the self-claim contract.

`GET /api/v1/trustees/payouts/:slug` no longer accepts
`destination_address` or `volunteer_invoice` query parameters. Those values
must eventually come from a server-frozen payout intent.

`POST /api/v1/trustees/payouts/:slug/sign` accepts only:

```json
{
  "layer1_psbt_signature_fragment": "placeholder text",
  "layer2_web_crypto_signature": "placeholder text"
}
```

Remove `trustee_public_key_hex`; the backend derives the stored signer key from
the authenticated task trustee relationship. These payout endpoints remain
non-production scaffolds: the backend does not yet build a PSBT, verify either
signature, or move funds. Do not present a successful response as a completed
or cryptographically approved payout.

## 4. Task state machines

Task responses now include `work_state_version` and
`financial_state_version`. They are monotonic state revision numbers. Use them
to notice stale display state, but do not send them back to choose or force a
transition.

The work lifecycle is:

```text
open -> in_progress -> pending_verification -> completed
```

The fundraising goal is a soft target. Reaching `goal_sats` does not start the
task and does not stop donations. Donations may exceed the target while
`financial_state` remains `ACTIVE`.

State actions use the authenticated cookie account as the actor. Never send an
actor, creator, volunteer, trustee, signer, current state, or target state in
JSON. Each retryable action requires a client-generated `Idempotency-Key`
header containing letters, digits, `.`, `_`, `:`, or `-`:

```http
POST /api/v1/tasks/:slug/start
Idempotency-Key: <unique key, maximum 128 characters>
Content-Type: application/json

{}
```

This creator-only action requires at least one approved volunteer and moves
`open` to `in_progress`.

```http
POST /api/v1/tasks/:slug/submit-for-verification
Idempotency-Key: <unique key, maximum 128 characters>
Content-Type: application/json

{ "reason": "All approved volunteers have submitted evidence." }
```

This creator-only action requires every approved volunteer to have at least
one submission and moves `in_progress` to `pending_verification`. Replace the
old frontend `POST /tasks/:slug/complete` call with this action where the
creator is requesting review; submitting evidence alone never changes the
task-wide state.

```http
POST /api/v1/tasks/:slug/verify
Idempotency-Key: <unique key, maximum 118 characters>
Content-Type: application/json

{}
```

This action requires a task trustee who is neither the creator nor a volunteer
on that task. It atomically moves work to `completed` and financial state from
`ACTIVE` to `LIQUIDATING`; new donation invoices then return `409`. The
onboarding flow described in step 5 is now complete, so expose verification
only to accounts whose roster status is `active`.

`reason` is optional on all three actions and is limited to 500 characters.
Send `{}` when it is omitted. A successful action returns:

```json
{
  "transitions": [
    {
      "id": 18,
      "state_kind": "work",
      "from_state": "open",
      "to_state": "in_progress",
      "version": 2,
      "actor_user_id": 12,
      "reason": "creator started task work",
      "created_at": "2026-07-18T19:40:00Z"
    }
  ],
  "replayed": false
}
```

Retrying the same action with the same key and body returns `200` with
`replayed: true`. Reusing a key for different action data, using a stale/illegal
transition, or failing readiness checks returns `409`. A non-owner or
non-independent verifier receives `403`.

State history is available newest first:

```http
GET /api/v1/tasks/:slug/state-history?state_kind=work&page=1&page_size=20
```

`state_kind` is optional and accepts `work` or `financial`. The response uses
the standard `pagination` object and a `transitions` array. Initial rows have
`from_state: null`; they are creation/migration baselines, not user actions.

The financial lifecycle exposed in task reads/history is:

```text
ACTIVE -> LIQUIDATING -> READY_FOR_PAYOUT -> PAYOUT_PROCESSING -> ARCHIVED
```

Only the completion boundary is currently connected. Later backend payout
services—not browser callers—will advance the remaining financial states.
`SYSTEM_LOCKDOWN` is reserved in the schema but has no executable transition
until entry, recovery, and authorization rules are approved.

## 5. Trustee onboarding

Replace the trustee self-claim screen with this sequence:

1. The creator sends `POST /api/v1/tasks/:slug/trustees/nominations` with
   `trustee_index`, the nominee's `user_id`, and optional `message`.
2. The invited account sends `POST /api/v1/tasks/:slug/trustees/accept` with no
   JSON identity. The response includes a 64-hex-character `proof_challenge`.
3. The trustee generates/loads their xpub and non-exportable P-256 browser key,
   signs the canonical proof below with both keys, then sends
   `POST /api/v1/tasks/:slug/trustees/keys`.

The canonical UTF-8 proof text is:

```text
pamojabuild:trustee-key-proof:v1:<task_slug>:<user_id>:<proof_challenge>:<xpub>:<web_crypto_pubkey_hex>
```

Hash that exact text once with SHA-256. Sign the digest with the private child
key corresponding to xpub path `m/0/0`; send the DER signature as lowercase hex
in `xpub_proof_signature_hex`. Export the uncompressed raw P-256 public key
(`04 || X || Y`) as hex in `web_crypto_pubkey_hex`. WebCrypto may return its
ECDSA signature as raw 64-byte `r || s`; send that hex in
`web_crypto_proof_signature_hex` (DER is also accepted). Send the challenge
unchanged. The backend validates the xpub against `BITCOIN_NETWORK`, which
defaults to `testnet3`; production must explicitly use `mainnet`.

Use `GET /api/v1/tasks/:slug/trustees` for the donor/creator roster. It returns
`{ "trustees": [...] }` with display name, user ID, slot, lifecycle status, and
timestamps. It deliberately omits xpubs, browser keys and proof material.

For an active trustee's rotation, first call
`POST /api/v1/tasks/:slug/trustees/keys/rotation-challenge`, sign the new key
pair using the returned challenge, then call
`POST /api/v1/tasks/:slug/trustees/keys/rotate` with the key fields plus a
required `reason`. Success is `204`.

Creator replacement is
`POST /api/v1/tasks/:slug/trustees/:index/replace` with `user_id`, required
`reason`, and optional `message`. It immediately removes the old account's
trustee authorization and leaves the successor `invited`; the successor must
complete the normal acceptance/proof sequence. Never present replacement as a
way to lower the future 3-of-5 payout threshold.

The current browser page keeps its non-exportable private key only in memory.
Move it to IndexedDB or integrate a reviewed external signer before presenting
later payout signing as durable. Never upload a private xpub or browser private
key.

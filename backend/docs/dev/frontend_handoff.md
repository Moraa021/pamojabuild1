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
current `POST /tasks/:slug/complete` caller also targets no backend route and
must not be invoked; completion belongs to the later state-machine/workflow
steps.

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

Do not send `user_id`. This remains an incomplete self-claim scaffold until
trustee nomination/onboarding is implemented.

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

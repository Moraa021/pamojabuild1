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
  endpoint to restore display state from the cookie. That endpoint is not
  implemented yet and should be added during API-contract work in step 3.

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

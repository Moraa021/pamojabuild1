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
  "token": "<jwt>",
  "user_id": 12,
  "is_admin": false,
  "display_name": "Amina",
  "expires_at": "2026-07-19T16:03:40Z"
}
```

Store `is_admin` only for genuine system-administration UI. Do not use it to display trustee or volunteer dashboards. Those dashboards will use task relationship data added by later workflow endpoints.

Request changes:

- `POST /api/v1/tasks`: remove `creator_id`; the backend uses the signed-in account.
- `POST /api/v1/tasks/:slug/apply`: send only `message`; remove `volunteer_id`.
- `POST /api/v1/tasks/:slug/trustees`: remove `user_id`; the backend uses the signed-in account. This route remains an incomplete scaffold until trustee onboarding is implemented.
- `POST /api/v1/auth/signout`: send the Bearer token, with no JSON body. A successful response is `204 No Content`. Clear the local session after success; also clear it if the token is already invalid.

Phone numbers must use international format beginning with `+`. Spaces, parentheses, and hyphens are accepted and normalized by the backend.

Trustee payout review/signing now returns `403 Forbidden` unless the signed-in account is a trustee for that task. The frontend should show a simple “You are not assigned as a trustee for this task” message and must not treat admin status as trustee approval.

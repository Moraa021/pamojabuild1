# Frontend Architecture Documentation

## API Integration Plan

| Feature | Endpoint | Method | Request Model | Response Model | Frontend Service |
|---|---|---|---|---|---|
| List tasks | `/api/v1/tasks` | GET | — | `TaskResponse[]` | `volunteerApi.listTasks()` |
| Create task | `/api/v1/tasks` | POST | `CreateTaskRequest` | `TaskResponse` | `taskApi.create()` |
| Get task | `/api/v1/tasks/:slug` | GET | — | `TaskResponse` | `taskApi.getBySlug()` |
| Register trustee keys | `/api/v1/tasks/:slug/trustees` | POST | `RegisterTrusteeKeysRequest` | — | `trusteeApi.registerKeys()` |
| Request invoice | `/api/v1/tasks/:slug/donate` | POST | `DonationRequest` | `DonationInvoiceResponse` | `lightningApi.requestInvoice()` |
| Get payout manifest | `/api/v1/trustees/payouts/:slug` | GET | — | `PayoutReviewResponse` | `escrowApi.getPayoutManifest()` |
| Submit co-signatures | `/api/v1/trustees/payouts/:slug/sign` | POST | `CoSignPayoutRequest` | `{threshold_reached}` | `escrowApi.submitCoSignatures()` |
| Apply to task | `/api/v1/tasks/:slug/apply` | POST | `{message}` | `VolunteerApplication` | `volunteerApi.applyToTask()` |
| Get applications | `/api/v1/volunteers/applications` | GET | — | `VolunteerApplication[]` | `volunteerApi.getApplications()` |
| Get profile | `/api/v1/volunteers/profile` | GET | — | `Volunteer` | `volunteerApi.getProfile()` |
| Update profile | `/api/v1/volunteers/profile` | PUT | `{display_name, bio}` | `Volunteer` | `volunteerApi.updateProfile()` |
| Create submission | `/api/v1/tasks/:slug/submissions` | POST | `{description, evidence_urls}` | `WorkSubmission` | `volunteerApi.createSubmission()` |
| Get submissions | `/api/v1/tasks/:slug/submissions` | GET | — | `WorkSubmission[]` | `volunteerApi.getSubmissions()` |
| Complete task | `/api/v1/tasks/:slug/complete` | POST | — | `TaskResponse` | `volunteerApi.completeTask()` |
| Get payments | `/api/v1/volunteers/payments` | GET | — | `VolunteerPayment[]` | `volunteerApi.getPayments()` |
| Save payment profile | `/api/v1/volunteers/payment-profile` | POST | `VolunteerPaymentProfile` | `VolunteerPaymentProfile` | `volunteerApi.savePaymentProfile()` |
| Get payment profile | `/api/v1/volunteers/payment-profile` | GET | — | `VolunteerPaymentProfile` | `volunteerApi.getPaymentProfile()` |

## Error Handling Strategy

- **Network errors**: `NetworkError` — shown via Toast, logged globally
- **Timeout (>15s)**: `TimeoutError` — auto-retry up to 3× with exponential backoff
- **401 Unauthorized**: redirect to sign-in, clear session
- **422 Validation**: field-level errors surfaced via `APIErrorDisplay`
- **5xx Server errors**: auto-retry 3×, then `APIErrorDisplay` with retry button

## Authentication Strategy

- JWT stored in `sessionStorage` (cleared on tab close)
- Injected into every request via `Authorization: Bearer <token>` header
- `setTokenAccessor()` registered in `authStore.js` wires the token to the API client
- Role stored alongside token; drives role-based nav via `ROLE_NAV` map

## Retry Strategy

- Retryable: HTTP 429 and 5xx
- Attempts: 3 (configurable via `ENV.RETRY_ATTEMPTS`)
- Delay: 800ms × 2^(attempt-1) — 800ms, 1.6s, 3.2s

## Caching Strategy

- No client-side caching beyond in-memory store state
- Stores are reset on navigation to avoid stale data
- CDN/service-worker caching deferred to deployment infrastructure

## Environment Configuration

Override `window.__ENV__` in `index.html` or via your build pipeline:

```js
window.__ENV__ = {
  API_BASE_URL:    'https://api.yourapp.com',
  API_VERSION:     'v1',
  REQUEST_TIMEOUT: 15000,
  RETRY_ATTEMPTS:  3,
  RETRY_DELAY_MS:  800,
  APP_ENV:         'production',
};
```

## Role–Nav Mapping

| Role | Nav Links |
|---|---|
| Guest | Browse campaigns, Sign in |
| Donor | Browse campaigns, My donations |
| Volunteer | Browse tasks, Dashboard, Applications, Payments |
| Creator | My campaigns, Create campaign |
| Trustee | Payout review, Register keys |
| Admin | Admin overview, All tasks, Trustee panel |
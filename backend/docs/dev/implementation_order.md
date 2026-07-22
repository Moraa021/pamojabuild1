# Backend Implementation Order

Last updated: 2026-07-22

This is the short continuity checklist for implementation. The detailed system reference is [app_flow_deep_dive.md](app_flow_deep_dive.md).

Status key: `NEXT`, `PENDING`, `DONE`, `BLOCKED`.

## Agreed product rules

- Users register general accounts; creator, volunteer, and trustee are task-specific relationships.
- A creator may assign themselves as a volunteer and may coordinate other volunteers.
- A trustee must never be a volunteer on the same task.
- A creator or volunteer cannot verify their own work or authorize their own payout.
- Creators nominate five task trustees; trustees must accept and register valid keys before donations begin.
- Trustee identities and relevant creator/volunteer relationships must be visible to donors.
- A task opens for volunteer applications immediately; the fundraising goal is informational and donations may exceed it.
- The creator starts work explicitly after at least one volunteer is approved.
- Every approved volunteer must submit evidence before the creator can request independent verification.

## Implementation order

1. **`DONE` — PostgreSQL and migrations:** The application is PostgreSQL-only, uses a clean versioned baseline with `golang-migrate` up/down migrations, never migrates during API startup, and runs database integration tests against PostgreSQL through `TEST_DATABASE_URL`.
2. **`DONE` — Accounts and authorization:** Accounts are general, admin is the only global capability, authentication uses revocable server-side sessions in secure HttpOnly cookies, actor IDs come from authentication, trustee payout routes require task membership, and PostgreSQL enforces the trustee/volunteer conflict.
3. **`DONE` — API contracts:** Registered routes use explicit snake_case DTOs, one safe error envelope, strict request decoding, service validation, intentional HTTP status codes, documented cookie authentication, paginated task filters, and `/auth/me`; frontend integration changes are recorded in `frontend_handoff.md`.
4. **`DONE` — Task state machines:** Work transitions are `open -> in_progress -> pending_verification -> completed`; creator and independent-trustee authorization, readiness checks, row locking/versioning, immutable history, idempotency, and atomic completion/donation closure are enforced. Financial transitions are linear and internal-only; `SYSTEM_LOCKDOWN` is reserved but deliberately unavailable until recovery rules are approved.
5. **`DONE` — Trustee onboarding:** Creators nominate five unique, non-conflicted task trustees; invitees accept before proving testnet/mainnet-matched xpub and browser-key ownership; only active memberships authorize trustee actions; the safe public roster omits key material; and replacement/rotation preserve history without lowering the future 3-of-5 payout threshold.
6. **`NEXT` — Volunteer workflow:** Implement self-assignment, application review/selection, transactional capacity, assignment-level progress, and evidence review; relationship mutations must share the task-row lock used by state readiness checks.
7. **`PENDING` — Lightning donations:** Complete invoice status/history behavior and harden settlement recovery, accounting delivery, and reconciliation; invoice creation already requires an existing `ACTIVE` task.
8. **`PENDING` — Ledger:** Add transactional, PostgreSQL-safe appends; unique references; explicit debits/credits; durable events; integrity checkpoints; and reconciliation.
9. **`PENDING` — Escrow and swaps:** Implement xpub validation, deterministic 3-of-5 vaults, derivation/UTXO records, and Lightning-to-on-chain swaps.
10. **`PENDING` — PSBT engine:** Implement immutable payout intents, PSBT construction, signer validation, signature collection, replay protection, and finalization.
11. **`PENDING` — Payout orchestrator:** Implement idempotent L1/L2 execution, partial-failure recovery, confirmations, ledger debits, volunteer payment records, and archival.
12. **`PENDING` — Production readiness:** Complete security review, monitoring, rate limiting, backups, disaster recovery, operational runbooks, and staged deployment.

## Working rules

- Complete and commit one cohesive, tested unit at a time using Conventional Commits.
- Update this checklist and the deep-dive after each completed unit.
- Document required frontend changes and integration contracts under `backend/docs/dev`; backend implementation remains the focus.

## Open product decisions and recommendations

These are not agreed rules yet. Resolve each one no later than the indicated
step so later schemas and APIs do not encode an accidental policy.

### One or three trustees for work verification

**Decision adopted in step 5:** One independent active trustee verifies work.
The 3-of-5 threshold applies only to later payout authorization.

**Recommendation:** Require one independent trustee to verify task completion,
while retaining 3-of-5 exclusively for payout authorization. Work verification
is an operational judgment and one verifier keeps small community tasks usable;
requiring three people for both verification and payout would add two separate
quorum waits to every task. The verifying trustee must not be the creator or a
volunteer on that task, and their identity, reason, and timestamp must remain
auditable.

**Architecture impact:** The current single transition actor is sufficient for
one verifier. A three-trustee rule would require a separate, immutable
`task_verification_votes` model, unique votes per trustee, quorum calculation,
vote withdrawal/replacement rules, and a concurrency-safe transition only when
the third valid vote is recorded. Do not model three votes by overwriting one
task status field.

### Whether work verification also approves payout

**Decision adopted in step 5:** Work verification and payout approval are
separate actions and records.

**Recommendation:** Keep work verification and payout approval separate.
Verification should confirm the work, complete the work lifecycle, close
donations, and enter `LIQUIDATING`. It should not release money or count as a
payout signature. Trustees should approve a later frozen payout intent only
after final balances, fees, recipients, amounts, and L1/L2 destinations are
known.

**Architecture impact:** Maintain separate records and permissions for work
verification and payout signatures. The eventual 3-of-5 payout approvals must
bind cryptographically to one immutable payout intent; a prior work-verification
action cannot safely approve financial details that did not yet exist.

### Whether a creator can remove a volunteer

**Decision needed:** At the start of step 6, before application approval,
capacity enforcement, assignment statuses, and completion-readiness queries are
implemented.

**Recommendation:** Allow removal of a nonresponsive volunteer before
`pending_verification`, but not as an unrecorded deletion or unrestricted way
to erase submitted work. Require a reason, preserve the original application
and assignment, record who acted and when, and notify the volunteer. Once a
volunteer has submitted work, unilateral removal should not discard their
evidence, attribution, or possible payment claim; use an explicit evidence
rejection/rework or later dispute policy instead.

Also allow a volunteer to withdraw. A task should calculate completion
readiness from active approved assignments, excluding properly withdrawn or
removed assignments. If no active volunteer remains, the creator must replace
one or use a separately designed task-cancellation path; the task must not
silently complete.

**Architecture impact:** Introduce assignment lifecycle fields or a dedicated
assignment/history model, for example `active`, `submitted`, `withdrawn`,
`removed`, and `replaced`, with actor, reason, and timestamps. Never delete the
application row. Assignment mutations and task readiness transitions must take
the same task-row lock so removal cannot race submission-for-verification.

### Whether a creator can reassign a volunteer

**Decision needed:** In step 6 alongside removal and transactional volunteer
caps.

**Recommendation:** Allow replacement before `pending_verification`. Implement
it as closing the old assignment and approving a new applicant, not changing
the user ID on an existing row. This gives creators a practical recovery path
when someone disappears while preserving a fair audit trail. A replacement
must satisfy the normal eligibility and trustee-conflict rules and must submit
their own work before verification can be requested.

**Architecture impact:** Capacity must count only active assignments, while
historical/replaced assignments remain queryable. Reassignment should be one
transaction under the task-row lock: mark the old assignment replaced, create
or activate the new assignment, recheck the volunteer cap and conflicts, and
append audit events. The API should expose a deliberate replacement action
rather than a generic assignment update endpoint.

### Whether final volunteer submission automatically requests verification

**Decision needed:** At the start of step 6, before submission finality,
evidence review, assignment statuses, and notification behavior are designed.

**Recommendation:** Do not move the task merely because every volunteer
uploaded something. Treat uploads as submitted evidence that the creator can
accept or return for rework. When the creator accepts the final outstanding
active assignment, automatically and atomically move the task to
`pending_verification`. This removes the redundant separate “request
verification” action without allowing an accidental or incomplete upload to
advance the whole task.

The creator still needs to know when evidence arrives. For the initial product,
use a durable in-app notification feed and clear dashboard counts rather than
SMS or a messaging provider. A phone number used for authentication is not
automatically consent to operational or promotional messages. Add SMS later
only if field testing shows in-app notifications are insufficient, with
explicit consent, delivery tracking, cost controls, retry handling, and an
approved provider.

**Architecture impact:** Step 6 needs assignment/submission review states and a
transaction that locks the task, accepts one submission, checks whether all
active assignments are accepted, and performs the state transition when the
last one clears. Notifications should be durable rows, not only in-memory
events, with read timestamps and idempotent event references. If external
messaging is added later, publish it through an outbox/worker so a provider
failure cannot roll back submission or state changes.

### Cancellation, failed tasks, unavailable trustees, and donor refunds

**Decision status:** Trustee liveness policy was adopted in step 5: replace an
unavailable trustee through the audited onboarding workflow and never lower the
3-of-5 threshold. The cryptographic recovery script remains a required step 9
decision before funds enter a vault. Define the user-facing cancellation/refund
policy before step 7 accepts production donations.

**Recommendation:** Support platform-triggered refunds when a task cannot
proceed, but do not let a creator immediately send money elsewhere or silently
cancel after work begins.

- Give recruitment a disclosed deadline. If no volunteer is active by that
  deadline, stop donations and enter a cancellation/refund review.
- Before work starts, allow the creator to request cancellation; after work
  starts, require independent review because volunteers may have performed work
  and earned a claim.
- If work fails or becomes impossible, preserve submissions and decisions,
  determine any legitimate incurred costs, and refund the remaining refundable
  balance under a recorded policy.
- If fewer than three trustees approve a payout, wait, notify, and use the
  trustee replacement/recovery process. Never silently lower the 3-of-5 payout
  threshold.
- Trustee availability must be checked before donations and before funds move
  into a vault. Replacing a trustee after funds are already in a 3-of-5 script
  cannot make the old vault spendable without enough existing keys. Step 9 must
  therefore define a reviewed recovery script or explicitly accept permanent
  lock risk before real funds are vaulted.

**Architecture impact:** The current linear work and financial graphs need
explicit cancellation/refund branches rather than overloading `ARCHIVED` or
`SYSTEM_LOCKDOWN`, for example cancellation requested/approved and
`REFUNDING -> REFUNDED`. Tasks need recruitment and execution deadlines.
Donations must be attributable to authenticated donor accounts and immutable
donation records; the current invoice record alone is insufficient for a
refund. Refund execution needs a donor-provided destination, amount and fee
policy, idempotency, ledger debits, retryable partial-failure handling, and
proof that the same donation cannot be refunded twice. L1 refunds must still
satisfy the vault's cryptographic spending policy.

### Whether an individual donor can request a refund

**Decision needed:** Set the policy and disclose it in the UI before step 7
accepts production donations. Implement any approved exceptional-refund path
alongside the refund/payout orchestrator in step 11.

**Recommendation:** Do not offer an unconditional change-of-mind refund after a
Lightning payment settles. Donations need enough finality for creators and
volunteers to rely on the available budget. An unpaid or expired invoice needs
no refund. A settled donation should become refund-eligible only when the task
is cancelled, the platform confirms a duplicate/incorrect charge, or a defined
fraud or operational-error policy applies.

A donor may submit a refund request for those exceptional cases, but submission
should not promise approval. The UI should clearly disclose finality,
cancellation conditions, treatment of unavoidable network/provider fees, and
the expected refund method before the donor pays.

**Architecture impact:** Store donor identity on each donation credit and add
auditable `refund_requests` and `refunds` records with reason, status, reviewer,
destination, approved amount, fee treatment, transaction/payment reference,
and idempotency key. Lightning does not provide a reusable “return address,” so
an approved Lightning refund normally requires a fresh invoice from the
authenticated donor. Refund reservations and execution must share the ledger
and orchestration controls used for payouts so refundable funds cannot also be
paid to volunteers.

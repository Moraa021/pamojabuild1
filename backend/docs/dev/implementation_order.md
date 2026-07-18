# Backend Implementation Order

Last updated: 2026-07-18

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
5. **`NEXT` — Trustee onboarding:** Implement nomination, acceptance, unique membership, key proof/validation, public roster data, replacement, and rotation.
6. **`PENDING` — Volunteer workflow:** Implement self-assignment, application review/selection, transactional capacity, assignment-level progress, and evidence review; relationship mutations must share the task-row lock used by state readiness checks.
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

**Decision needed:** During step 5, before trustee onboarding assigns
capabilities and the frontend presents trustee actions. It must be final before
step 6 connects evidence review to completion.

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

**Decision needed:** Confirm the separation during step 5 when trustee duties
are defined; it becomes irreversible design input in steps 9 and 10 when the
payout intent and PSBT approval model are created.

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

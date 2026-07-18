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

## Implementation order

1. **`NEXT` — PostgreSQL and migrations:** Remove SQLite completely, replace the legacy migrations with a clean PostgreSQL baseline, adopt `golang-migrate` up/down migrations, remove automatic server-startup migration execution, and test repositories against PostgreSQL.
2. **`PENDING` — Accounts and authorization:** Replace permanent creator/volunteer/trustee roles with task relationships, retain only genuine global capabilities such as admin, derive actor IDs from authentication, and enforce ownership and conflict rules.
3. **`PENDING` — API contracts:** Standardize request/response DTOs, errors, validation, and OpenAPI documentation.
4. **`PENDING` — Task state machines:** Enforce work and financial transitions with authorization, conditional updates, history, idempotency, and recovery rules.
5. **`PENDING` — Trustee onboarding:** Implement nomination, acceptance, unique membership, key proof/validation, public roster data, replacement, and rotation.
6. **`PENDING` — Volunteer workflow:** Implement self-assignment, applications, selection, capacity, submissions, evidence review, verification, and conflict-safe completion.
7. **`PENDING` — Lightning donations:** Require eligible active tasks, complete invoice status/history behavior, and harden settlement recovery and reconciliation.
8. **`PENDING` — Ledger:** Add transactional, PostgreSQL-safe appends; unique references; explicit debits/credits; durable events; integrity checkpoints; and reconciliation.
9. **`PENDING` — Escrow and swaps:** Implement xpub validation, deterministic 3-of-5 vaults, derivation/UTXO records, and Lightning-to-on-chain swaps.
10. **`PENDING` — PSBT engine:** Implement immutable payout intents, PSBT construction, signer validation, signature collection, replay protection, and finalization.
11. **`PENDING` — Payout orchestrator:** Implement idempotent L1/L2 execution, partial-failure recovery, confirmations, ledger debits, volunteer payment records, and archival.
12. **`PENDING` — Production readiness:** Complete security review, monitoring, rate limiting, backups, disaster recovery, operational runbooks, and staged deployment.

## Working rules

- Complete and commit one cohesive, tested unit at a time using Conventional Commits.
- Update this checklist and the deep-dive after each completed unit.
- Document required frontend changes and integration contracts under `backend/docs/dev`; backend implementation remains the focus.

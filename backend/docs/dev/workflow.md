# PamojaBuild Backend Development Workflow

## Objective
Build a production-grade, multi-tenant Bitcoin and Lightning crowdfunding platform without creating architectural debt, cross-team blockers, or security bottlenecks.

The primary goal is to ensure every engineering team can work independently while maintaining strict cryptographic guarantees.

---

## Phase 0: Architecture Freeze
* **Duration:** 1 Week
* **Rule:** No coding allowed.
* **Deliverables:**
    * Domain model
    * State machine definitions
    * API contracts
    * Database schema
    * Security model
    * Event model

### Questions that must be answered before implementation:

#### Task Lifecycle
`ACTIVE` → `LIQUIDATING` → `READY_FOR_PAYOUT` → `PAYOUT_PROCESSING` → `ARCHIVED`

Define:
* Allowed transitions
* Invalid transitions
* Recovery procedures
* Failure handling

#### Trust Model
Define:
* Trustee onboarding
* Trustee removal
* Trustee replacement
* Trustee key rotation

#### Bitcoin Model
Define:
* Address derivation strategy
* Xpub storage policy
* PSBT lifecycle
* Signing requirements

#### Lightning Model
Define:
* Invoice creation
* Invoice settlement
* Donation reconciliation
* Failed payment handling

* **Output:** Approved Architecture Specification v1.
* **Rule:** No development begins until approved.

---

## Phase 1: API Contract First Development
* **Duration:** 1 Week
* **Process:** Backend and frontend jointly define every endpoint.

### Example Endpoints:
* `POST /tasks`
* `POST /tasks/:id/donate`
* `POST /tasks/:id/trustees`
* `GET /tasks/:id/payout`
* `POST /tasks/:id/sign`

### Every endpoint must have:
* Request schema
* Response schema
* Error schema
* Validation rules

* **Output:** OpenAPI Specification (Swagger) — This becomes the source of truth.

---

## Phase 2: Domain Module Separation
The backend is divided into independent domains. No domain can directly access another domain's database tables.

### Modules:
* **Task Service**
    * *Responsibilities:* Task lifecycle, State transitions
* **Trustee Service**
    * *Responsibilities:* Trustee registration, Signature verification
* **Ledger Service**
    * *Responsibilities:* Immutable accounting, Balance calculations
* **Lightning Service**
    * *Responsibilities:* LND communication, Invoice lifecycle
* **Escrow Service**
    * *Responsibilities:* Xpub management, Multisig generation
* **Payout Service**
    * *Responsibilities:* PSBT assembly, Distribution orchestration
* **Audit Service**
    * *Responsibilities:* HMAC verification, Compliance logs

* **Rule:** Domains communicate only through events. Never through direct database access.

---

## Phase 3: Database Before Bitcoin
* **Duration:** 2 Weeks
* **Implementation:** Implement tables/collections for `tasks`, `trustees`, `ledger_entries`, `task_states`, and `audit_logs` without Bitcoin integration.
* **Objectives:**
    * Verify workflow correctness
    * Verify state machine
    * Verify permissions
* **Constraints:** No sats moved. No invoices created. No PSBTs generated. Only business logic.

---

## Phase 4: Event Driven Architecture
Introduce message bus.

### Events:
* `TaskCreated`
* `DonationReceived`
* `ThresholdReached`
* `TaskLiquidating`
* `TaskReadyForPayout`
* `TrusteeSigned`
* `PayoutCompleted`

* **Rule:** Services react to events. Services never call each other directly.
* **Benefits:** Reduced coupling, Easier testing, Easier scaling.

---

## Phase 5: Lightning Integration
* **Duration:** 2 Weeks
* **Process:** Build Lightning in isolation.
* **Requirements:**
    * Invoice Generation
    * Invoice Settlement Listener
    * Donation Attribution
    * Ledger Synchronization
* **Tests:**
    * Payment success
    * Payment timeout
    * Duplicate settlement
    * Node restart recovery
* **Constraint:** Lightning must be production-ready before touching escrow.

---

## Phase 6: Escrow Service
* **Duration:** 3 Weeks
* **Process:** Build independently from Lightning.
* **Capabilities:**
    * Xpub registration
    * Child key derivation
    * Multisig script generation
* **Tests:**
    * Address generation
    * Deterministic derivation
    * Trustee replacement
    * Key rotation
    * Address recovery
* **Constraint:** No payouts yet. Only vault creation.

---

## Phase 7: PSBT Engine
* **Duration:** 3 Weeks
* **Process:** Build as a standalone subsystem.
* **Capabilities:**
    * Transaction assembly
    * PSBT creation
    * Signature collection
    * Signature validation
    * Transaction finalization
* **Tests:**
    * Invalid signature
    * Duplicate signer
    * Missing signer
    * Threshold reached
* **Output:** Battle-tested signing engine.

---

## Phase 8: Unified Payout Orchestrator
Only now connect: `Lightning Service` + `Escrow Service` + `Trustee Service`.

### Workflow:
1. Verify task state
2. Verify ledger integrity
3. Verify HMAC chain
4. Verify trustee signatures
5. Finalize PSBT
6. Broadcast transaction
7. Release Lightning balance
8. Archive task

* **Rule:** Single orchestration service owns this process. No other service can trigger payouts.

---

## Phase 9: Security Review
Required before production launch.

### Review Focus Areas:
* Authentication
* Authorization
* Key storage
* Trustee impersonation
* Replay attacks
* Invoice spoofing
* Database tampering
* API abuse
* Rate limiting

* **Process:** Conduct threat modeling. No deployment before passing review.

---

## Phase 10: Production Readiness

### Requirements:
* Monitoring
* Metrics
* Audit logging
* Backup strategy
* Disaster recovery
* Key recovery procedures

### Deployment Checklist:
* Testnet
* Staging
* Canary
* Production

* **Rule:** No direct production releases. All releases pass staging first.

---

## Golden Rules
1. Bitcoin code never touches business logic.
2. Business logic never touches cryptographic primitives directly.
3. Every satoshi movement creates a ledger event.
4. State transitions are the only source of truth.
5. No service owns another service's database tables.
6. All critical actions are idempotent.
7. Every payout must be reproducible from audit logs.
8. Every subsystem must be deployable independently.
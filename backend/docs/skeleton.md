To maintain strict domain boundaries where no module touches another module's database tables, the application will be separated into five distinct, decoupled packages. Communication between these packages will be driven via standard interfaces and decoupled event/service calls.

For event-driven behavior, see `docs/events.md` for event topics and payload definitions.

---

### Package 1: `task` (Campaign & Volunteer Social Logic)

This domain owns the human facing workflow, social descriptions, and volunteer metadata. It drives the core task data structures.

#### 1.1 API Payloads (`internal/task/delivery/http/payloads.go`)

```go
package http

import "time"

type CreateTaskRequest struct {
	CreatorID      int64  `json:"creator_id" binding:"required"`
	Title          string `json:"title" binding:"required"`
	Description    string `json:"description" binding:"required"`
	Category       string `json:"category" binding:"required"`
	Region         string `json:"region" binding:"required"`
	LocationDetail string `json:"location_detail,omitempty"`
	GoalSats       int64  `json:"goal_sats,omitempty"`
	MaxVolunteers  int64  `json:"max_volunteers"`
	VolunteerMode  string `json:"volunteer_mode" binding:"required"` // "open" or "approval_required"
}

type TaskResponse struct {
	ID             int64     `json:"id"`
	Slug           string    `json:"slug"`
	CreatorID      int64     `json:"creator_id"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Category       string    `json:"category"`
	Region         string    `json:"region"`
	LocationDetail string    `json:"location_detail,omitempty"`
	Status         string    `json:"status"`           // "open", "in_progress", "pending_verification", "completed"
	FinancialState string    `json:"financial_state"`  // "ACTIVE", "LIQUIDATING", "READY_FOR_PAYOUT", "SYSTEM_LOCKDOWN", "ARCHIVED"
	GoalSats       int64     `json:"goal_sats,omitempty"`
	MaxVolunteers  int64     `json:"max_volunteers"`
	VolunteerMode  string    `json:"volunteer_mode"`
	ImagePath      string    `json:"image_path,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

```

#### 1.2 Core Interfaces (`internal/task/domain.go`)

```go
package task

import (
	"context"
	"time"
)

type Task struct {
	ID             int64
	Slug           string
	CreatorID      int64
	Title          string
	Description    string
	Category       string
	Region         string
	LocationDetail string
	Status         string
	FinancialState string
	GoalSats       int64
	MaxVolunteers  int64
	VolunteerMode  string
	ImagePath      string
	CreatedAt      time.Time
}

type Repository interface {
	Create(ctx context.Context, t *Task) error
	GetByID(ctx context.Context, id int64) (*Task, error)
	GetBySlug(ctx context.Context, slug string) (*Task, error)
	UpdateStatus(ctx context.Context, slug string, status string) error
	UpdateFinancialState(ctx context.Context, slug string, state string) error
}

type Service interface {
	CreateCampaign(ctx context.Context, req *Task) (*Task, error)
	TransitionVolunteerStatus(ctx context.Context, slug string, targetStatus string) error
	TransitionFinancialState(ctx context.Context, slug string, targetState string) error
}

```

---

### Package 2: `trustee` (Onboarding, Keys & Multi-Sig Membership)

This domain owns human authentication, the assignment of the 5 trustee slots, and the public keys used to authorize payouts.

#### 2.1 API Payloads (`internal/trustee/delivery/http/payloads.go`)

```go
package http

type RegisterTrusteeKeysRequest struct {
	UserID             int64  `json:"user_id" binding:"required"`
	TrusteeIndex       int32  `json:"trustee_index" binding:"required"` // Strict range 0-4
	Xpub               string `json:"xpub" binding:"required"`               // BIP32 HD Master Public Key
	WebCryptoPubkeyHex string `json:"web_crypto_pubkey_hex" binding:"required"` // Browser-generated public key
}

```

#### 2.2 Core Interfaces (`internal/trustee/domain.go`)

```go
package trustee

import "context"

type User struct {
	ID           int64
	Email        string
	PasswordHash string
	DisplayName  string
}

type TrusteeKey struct {
	TaskSlug           string
	TrusteeIndex       int32
	UserID             int64
	Xpub               string
	WebCryptoPubkeyHex string
}

type UserRepository interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id int64) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
}

type KeyRepository interface {
	SaveKeys(ctx context.Context, key *TrusteeKey) error
	GetKeysByTask(ctx context.Context, taskSlug string) ([]TrusteeKey, error)
	GetSpecificTrustee(ctx context.Context, taskSlug string, trusteeIndex int32) (*TrusteeKey, error)
}

type Service interface {
	RegisterUser(ctx context.Context, email, password, displayName string) (*User, error)
	AssignTrusteeSlot(ctx context.Context, slug string, key *TrusteeKey) error
	VerifyWebCryptoSignature(ctx context.Context, pubKeyHex string, message []byte, signatureHex string) (bool, error)
}

```

---

### Package 3: `lightning` (Layer 2 Payment Ingestion)

This domain isolates all LND interactions. It handles raw invoice generation and intercepts asynchronous payment arrivals.

#### 3.1 API Payloads (`internal/lightning/delivery/http/payloads.go`)

```go
package http

type DonationRequest struct {
	AmountSats int64 `json:"amount_sats" binding:"required,gt=0"`
}

type DonationInvoiceResponse struct {
	PaymentRequest string `json:"payment_request"` // The BOLT11 raw string text for the QR code
	PaymentHash    string `json:"payment_hash"`    // Hex identifier string to poll settlement status
	ExpiresAt      int64  `json:"expires_at"`      // Unix timestamp cutoff
}

```

#### 3.2 Core Interfaces (`internal/lightning/domain.go`)

```go
package lightning

import (
	"context"
	"time"
)

type Invoice struct {
	PaymentRequest string
	PaymentHash    string
	AmountSats     int64
	TaskSlug       string
	Settled        bool
	SettledAt      time.Time
}

type Client interface {
	GenerateBolt11Invoice(ctx context.Context, taskSlug string, amountSats int64) (*Invoice, error)
	SubscribeInvoiceSettlements(ctx context.Context, callback func(settledInvoice *Invoice)) error
}

type Service interface {
	RequestDonationInvoice(ctx context.Context, taskSlug string, amountSats int64) (*Invoice, error)
	ProcessIncomingSettlement(ctx context.Context, invoice *Invoice) error
}

```

---

### Package 4: `ledger` (Chained Audit Book & Anti-Tampering Engine)

The ledger owns the single source of financial balance truth. It performs the cryptographic append-only HMAC chaining routines.

#### 4.1 Core Interfaces (`internal/ledger/domain.go`)

```go
package ledger

import "context"

type LedgerEntry struct {
	ID           int64
	TaskSlug     string
	EntryType    string // "INBOUND_DONATION", "SUBMARINE_SWAP", "TAIL_PAYOUT"
	AmountSats   int64
	ReferenceID  string // LND payment hash or L1 TxID
	PreviousHash []byte
	RowHMAC      []byte
}

type BalanceSummary struct {
	L2BalanceSats int64
	L1BalanceSats int64
	CurrentIndex  int32
}

type Repository interface {
	GetLastEntry(ctx context.Context, taskSlug string) (*LedgerEntry, error)
	AppendEntry(ctx context.Context, entry *LedgerEntry) error
	GetTaskBalance(ctx context.Context, taskSlug string) (*BalanceSummary, error)
	UpdateBalances(ctx context.Context, taskSlug string, l2Delta, l1Delta int64) error
	IncrementDerivationIndex(ctx context.Context, taskSlug string) error
}

type SecurityService interface {
	CalculateRowHMAC(entry *LedgerEntry, previousHash []byte, serverSecret string) ([]byte, error)
	VerifyEntireChainIntegrity(ctx context.Context, taskSlug string, serverSecret string) (bool, error)
	RecordValidatedTransaction(ctx context.Context, taskSlug string, entryType string, amountSats int64, refID string) error
}

```

---

### Package 5: `escrow` (Multi-Sig Addressing & Final Payout Orchestration)

This package holds the layout for multi-sig generation and coordinates the trustee multi-signature co-signing flows outlined in the specification guide.

#### 5.1 API Payloads (`internal/escrow/delivery/http/payloads.go`)

```go
package http

type PayoutReviewResponse struct {
	TaskSlug          string `json:"task_slug"`
	UnsignedPsbtHex   string `json:"unsigned_psbt_hex"`   // Raw text representation for Layer 1 hardware
	VolunteerInvoice  string `json:"volunteer_invoice"`  // Raw invoice text string for Layer 2 tail balance
	L1AmountSats      int64  `json:"l1_amount_sats"`
	L2AmountSats      int64  `json:"l2_amount_sats"`
}

type CoSignPayoutRequest struct {
	TrusteePublicKeyHex         string `json:"trustee_public_key_hex" binding:"required"`
	Layer1PsbtSignatureFragment string `json:"layer1_psbt_signature_fragment" binding:"required"`
	Layer2WebCryptoSignature     string `json:"layer2_web_crypto_signature" binding:"required"`
}

```

#### 5.2 Core Interfaces (`internal/escrow/domain.go`)

```go
package escrow

import "context"

type SignatureCollection struct {
	TaskSlug            string
	TrusteePublicKeyHex string
	L1SignatureFragment string
	L2WebCryptoSignature string
}

type AddressDerivationService interface {
	Derive3Of5MultiSigAddress(xpubs []string, index uint32) (string, error)
}

type PayoutOrchestrator interface {
	PreparePayoutManifest(ctx context.Context, taskSlug string, destinationAddress string, volunteerInvoice string) (*SignatureCollection, error)
	SubmitTrusteeSignature(ctx context.Context, taskSlug string, payload *SignatureCollection) (bool, error) // Returns true if 3/5 threshold is reached
	FinalizeAndBroadcastPayout(ctx context.Context, taskSlug string) error
}

```

---

### Expected API Endpoint Routing Map

To help your frontend team coordinate, they can code their networking files against these explicit, structured routes that match the structures above:

```text
// Campaign Social Paths
POST /api/v1/tasks              -> http.CreateTaskHandler (Payload: CreateTaskRequest)
GET  /api/v1/tasks/:task_slug   -> http.GetTaskHandler

// Trustee Setup Path
POST /api/v1/tasks/:task_slug/trustees -> http.RegisterTrusteeKeysHandler (Payload: RegisterTrusteeKeysRequest)

// Inbound Donor Path
POST /api/v1/tasks/:task_slug/donate   -> http.RequestDonationInvoiceHandler (Payload: DonationRequest)

// Trustee Signoff Review Dashboard Paths
GET  /api/v1/trustees/payouts/:task_slug      -> http.GetPayoutReviewManifestHandler
POST /api/v1/trustees/payouts/:task_slug/sign -> http.SubmitCoSignaturesHandler (Payload: CoSignPayoutRequest)

```
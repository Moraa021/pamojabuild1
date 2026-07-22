package trustee

import (
	"context"
	"errors"
	"time"
)

var (
	ErrRegistrationConflict = errors.New("trustee registration conflict")
	ErrTaskNotFound         = errors.New("trustee task not found")
	ErrAssignmentNotFound   = errors.New("trustee assignment not found")
	ErrUserNotFound         = errors.New("trustee nominee not found")
	ErrInvalidState         = errors.New("trustee assignment state conflict")
	ErrNotTaskCreator       = errors.New("only the task creator may manage trustees")
)

type AssignmentStatus string

const (
	StatusInvited  AssignmentStatus = "invited"
	StatusAccepted AssignmentStatus = "accepted"
	StatusActive   AssignmentStatus = "active"
	StatusRevoked  AssignmentStatus = "revoked"
	StatusReplaced AssignmentStatus = "replaced"
)

// TrusteeKey is retained as the active-key projection consumed by the escrow
// scaffolding. Onboarding writes it only after both ownership proofs pass.
type TrusteeKey struct {
	TaskSlug           string
	TrusteeIndex       int32
	UserID             int64
	Xpub               string
	WebCryptoPubkeyHex string
}

type Assignment struct {
	TaskSlug          string
	TrusteeIndex      int32
	UserID            int64
	DisplayName       string
	Status            AssignmentStatus
	NominatedBy       int64
	NominationMessage string
	ProofChallenge    string
	InvitedAt         time.Time
	AcceptedAt        *time.Time
	ActivatedAt       *time.Time
	EndedAt           *time.Time
}

type KeyRegistration struct {
	TaskSlug                   string
	TrusteeIndex               int32
	UserID                     int64
	Xpub                       string
	WebCryptoPubkeyHex         string
	XpubProofSignatureHex      string
	WebCryptoProofSignatureHex string
	ProofChallenge             string
	RotationReason             string
}

type Replacement struct {
	TaskSlug       string
	TrusteeIndex   int32
	OldUserID      int64
	NewUserID      int64
	ActorUserID    int64
	Reason         string
	Message        string
	ProofChallenge string
}

type Repository interface {
	Nominate(ctx context.Context, assignment *Assignment) error
	Accept(ctx context.Context, taskSlug string, userID int64, proofChallenge string) (*Assignment, error)
	Activate(ctx context.Context, registration *KeyRegistration) (*Assignment, error)
	IssueRotationChallenge(ctx context.Context, taskSlug string, userID int64, proofChallenge string) error
	RotateKeys(ctx context.Context, registration *KeyRegistration, actorUserID int64) error
	Replace(ctx context.Context, replacement *Replacement) error
	GetAssignmentForUser(ctx context.Context, taskSlug string, userID int64) (*Assignment, error)
	GetRoster(ctx context.Context, taskSlug string) ([]Assignment, error)
}

// KeyRepository is the narrow read interface used by escrow and payout
// scaffolding. It deliberately exposes active keys, not invitation records.
type KeyRepository interface {
	GetKeysByTask(ctx context.Context, taskSlug string) ([]TrusteeKey, error)
	GetSpecificTrustee(ctx context.Context, taskSlug string, trusteeIndex int32) (*TrusteeKey, error)
}

type Service interface {
	Nominate(ctx context.Context, taskSlug string, actorUserID int64, trusteeIndex int32, userID int64, message string) (*Assignment, error)
	Accept(ctx context.Context, taskSlug string, actorUserID int64) (*Assignment, error)
	RegisterKeys(ctx context.Context, taskSlug string, actorUserID int64, registration *KeyRegistration) (*Assignment, error)
	PrepareKeyRotation(ctx context.Context, taskSlug string, actorUserID int64) (string, error)
	RotateKeys(ctx context.Context, taskSlug string, actorUserID int64, registration *KeyRegistration) error
	Replace(ctx context.Context, taskSlug string, actorUserID int64, trusteeIndex int32, newUserID int64, reason, message string) error
	GetTaskTrustees(ctx context.Context, taskSlug string) ([]Assignment, error)
}

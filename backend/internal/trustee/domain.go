package trustee

import (
	"context"
	"errors"
)

var (
	ErrRegistrationConflict = errors.New("trustee registration conflict")
	ErrTaskNotFound         = errors.New("trustee task not found")
)

type TrusteeKey struct {
	TaskSlug           string
	TrusteeIndex       int32
	UserID             int64
	Xpub               string
	WebCryptoPubkeyHex string
}

type KeyRepository interface {
	SaveKeys(ctx context.Context, key *TrusteeKey) error
	GetKeysByTask(ctx context.Context, taskSlug string) ([]TrusteeKey, error)
	GetSpecificTrustee(ctx context.Context, taskSlug string, trusteeIndex int32) (*TrusteeKey, error)
}

type Service interface {
	AssignTrusteeSlot(ctx context.Context, slug string, key *TrusteeKey) error
	VerifyWebCryptoSignature(ctx context.Context, pubKeyHex string, message []byte, signatureHex string) (bool, error)
	GetTaskTrustees(ctx context.Context, taskSlug string) ([]TrusteeKey, error)
}

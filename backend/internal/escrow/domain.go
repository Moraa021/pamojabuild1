package escrow

import "context"

type SignatureCollection struct {
	TaskSlug             string
	TrusteePublicKeyHex  string
	L1SignatureFragment  string
	L2WebCryptoSignature string
}

type PayoutManifest struct {
	TaskSlug         string
	UnsignedPSBTHex  string
	VolunteerInvoice string
	L1AmountSats     int64
	L2AmountSats     int64
}

type AddressDerivationService interface {
	Derive3Of5MultiSigAddress(xpubs []string, index uint32) (string, error)
}

type PayoutOrchestrator interface {
	PreparePayoutManifest(ctx context.Context, taskSlug string) (*PayoutManifest, error)
	SubmitTrusteeSignature(ctx context.Context, taskSlug string, trusteeUserID int64, l1Signature, l2Signature string) (bool, error)
	FinalizeAndBroadcastPayout(ctx context.Context, taskSlug string) error
}

type SignatureRepository interface {
	SaveSignature(ctx context.Context, sig *SignatureCollection) error
	GetSignatures(ctx context.Context, taskSlug string) ([]SignatureCollection, error)
	GetSignatureCount(ctx context.Context, taskSlug string) (int, error)
}

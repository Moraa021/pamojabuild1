package escrow

import "context"

type SignatureCollection struct {
	TaskSlug               string
	TrusteePublicKeyHex    string
	L1SignatureFragment    string
	L2WebCryptoSignature   string
}

type AddressDerivationService interface {
	Derive3Of5MultiSigAddress(xpubs []string, index uint32) (string, error)
}

type PayoutOrchestrator interface {
	PreparePayoutManifest(ctx context.Context, taskSlug string, destinationAddress string, volunteerInvoice string) (*SignatureCollection, error)
	SubmitTrusteeSignature(ctx context.Context, taskSlug string, payload *SignatureCollection) (bool, error)
	FinalizeAndBroadcastPayout(ctx context.Context, taskSlug string) error
}

type SignatureRepository interface {
	SaveSignature(ctx context.Context, sig *SignatureCollection) error
	GetSignatures(ctx context.Context, taskSlug string) ([]SignatureCollection, error)
	GetSignatureCount(ctx context.Context, taskSlug string) (int, error)
}

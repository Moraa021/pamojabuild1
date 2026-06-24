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

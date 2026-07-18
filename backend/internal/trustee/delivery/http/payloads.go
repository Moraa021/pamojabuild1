package http

type RegisterTrusteeKeysRequest struct {
	TrusteeIndex       int32  `json:"trustee_index" binding:"min=0,max=4"`      // Strict range 0-4
	Xpub               string `json:"xpub" binding:"required"`                  // BIP32 HD Master Public Key
	WebCryptoPubkeyHex string `json:"web_crypto_pubkey_hex" binding:"required"` // Browser-generated public key
}

package http

type RegisterTrusteeKeysRequest struct {
	TrusteeIndex       int32  `json:"trustee_index" binding:"min=0,max=4"`
	Xpub               string `json:"xpub" binding:"required,max=255"`
	WebCryptoPubkeyHex string `json:"web_crypto_pubkey_hex" binding:"required,max=512"`
}

type TrusteeRegistrationResponse struct {
	TaskSlug     string `json:"task_slug"`
	TrusteeIndex int32  `json:"trustee_index"`
	UserID       int64  `json:"user_id"`
}

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

package http

type DonationRequest struct {
	AmountSats int64 `json:"amount_sats" binding:"required,gt=0"`
}

type DonationInvoiceResponse struct {
	PaymentRequest string `json:"payment_request"` // The BOLT11 raw string text for the QR code
	PaymentHash    string `json:"payment_hash"`    // Hex identifier string to poll settlement status
	ExpiresAt      int64  `json:"expires_at"`      // Unix timestamp cutoff
}

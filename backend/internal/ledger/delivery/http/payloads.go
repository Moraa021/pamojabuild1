package http

type LedgerTransactionRequest struct {
	EntryType   string `json:"entry_type" binding:"required"`
	AmountSats  int64  `json:"amount_sats" binding:"required"`
	ReferenceID string `json:"reference_id" binding:"required"`
}

type VerifyChainResponse struct {
	TaskSlug string `json:"task_slug"`
	Valid    bool   `json:"valid"`
}

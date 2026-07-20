package http

type VerifyChainResponse struct {
	TaskSlug string `json:"task_slug"`
	Valid    bool   `json:"valid"`
}

type TaskBalanceResponse struct {
	L2BalanceSats int64 `json:"l2_balance_sats"`
	L1BalanceSats int64 `json:"l1_balance_sats"`
	CurrentIndex  int32 `json:"current_index"`
}

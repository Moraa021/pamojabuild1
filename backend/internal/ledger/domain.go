package ledger

import "context"

type LedgerEntry struct {
	ID           int64
	TaskSlug     string
	EntryType    string // "INBOUND_DONATION", "SUBMARINE_SWAP", "TAIL_PAYOUT"
	AmountSats   int64
	ReferenceID  string // LND payment hash or L1 TxID
	PreviousHash []byte
	RowHMAC      []byte
}

type BalanceSummary struct {
	L2BalanceSats int64
	L1BalanceSats int64
	CurrentIndex  int32
}

type Repository interface {
	GetLastEntry(ctx context.Context, taskSlug string) (*LedgerEntry, error)
	AppendEntry(ctx context.Context, entry *LedgerEntry) error
	GetTaskBalance(ctx context.Context, taskSlug string) (*BalanceSummary, error)
	UpdateBalances(ctx context.Context, taskSlug string, l2Delta, l1Delta int64) error
	IncrementDerivationIndex(ctx context.Context, taskSlug string) error
}

type SecurityService interface {
	CalculateRowHMAC(entry *LedgerEntry, previousHash []byte, serverSecret string) ([]byte, error)
	VerifyEntireChainIntegrity(ctx context.Context, taskSlug string, serverSecret string) (bool, error)
	RecordValidatedTransaction(ctx context.Context, taskSlug string, entryType string, amountSats int64, refID string) error
}

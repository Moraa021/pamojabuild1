package lightning

import (
	"context"
	"time"
)

const (
	InvoiceStatusPending = "pending"
	InvoiceStatusSettled = "settled"
	InvoiceStatusExpired = "expired"
)

type Invoice struct {
	PaymentRequest string
	PaymentHash    string
	AmountSats     int64
	TaskSlug       string
	Status         string
	Settled        bool
	CreatedAt      time.Time
	ExpiresAt      time.Time
	SettledAt      time.Time
	AddIndex       int64
	SettleIndex    int64
}

type InvoiceRequest struct {
	TaskSlug   string
	AmountSats int64
	Memo       string
	Expiry     time.Duration
}

type SettlementHandler func(ctx context.Context, settledInvoice *Invoice) error

type NodeClient interface {
	CreateInvoice(ctx context.Context, request InvoiceRequest) (*Invoice, error)
	SubscribeInvoiceSettlements(ctx context.Context, sinceSettleIndex int64, handler SettlementHandler) error
}

type Repository interface {
	SaveInvoice(ctx context.Context, invoice *Invoice) error
	GetByPaymentHash(ctx context.Context, paymentHash string) (*Invoice, error)
	MarkSettled(ctx context.Context, paymentHash string, settledAt time.Time, settleIndex int64) (bool, error)
}

type Service interface {
	RequestDonationInvoice(ctx context.Context, taskSlug string, amountSats int64) (*Invoice, error)
	ProcessIncomingSettlement(ctx context.Context, invoice *Invoice) error
}

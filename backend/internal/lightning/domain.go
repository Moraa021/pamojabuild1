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

type Client interface {
	GenerateBolt11Invoice(ctx context.Context, taskSlug string, amountSats int64) (*Invoice, error)
	SubscribeInvoiceSettlements(ctx context.Context, callback func(settledInvoice *Invoice)) error
	SaveInvoice(ctx context.Context, invoice *Invoice) error
	GetByPaymentHash(ctx context.Context, paymentHash string) (*Invoice, error)
	MarkSettled(ctx context.Context, paymentHash string, settledAt time.Time, settleIndex int64) (bool, error)
}

type Service interface {
	RequestDonationInvoice(ctx context.Context, taskSlug string, amountSats int64) (*Invoice, error)
	ProcessIncomingSettlement(ctx context.Context, invoice *Invoice) error
}

package lightning

import (
	"context"
	"time"
)

type Invoice struct {
	PaymentRequest string
	PaymentHash    string
	AmountSats     int64
	TaskSlug       string
	Settled        bool
	SettledAt      time.Time
}

type Client interface {
	GenerateBolt11Invoice(ctx context.Context, taskSlug string, amountSats int64) (*Invoice, error)
	SubscribeInvoiceSettlements(ctx context.Context, callback func(settledInvoice *Invoice)) error
	SaveInvoice(ctx context.Context, invoice *Invoice) error
	GetByPaymentHash(ctx context.Context, paymentHash string) (*Invoice, error)
	UpdateSettlement(ctx context.Context, paymentHash string, settledAt time.Time) error
}

type Service interface {
	RequestDonationInvoice(ctx context.Context, taskSlug string, amountSats int64) (*Invoice, error)
	ProcessIncomingSettlement(ctx context.Context, invoice *Invoice) error
}

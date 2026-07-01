package service

import (
    "context"
    "errors"
    "testing"
    "time"

    "pamojabuild1/backend/internal/config"
    "pamojabuild1/backend/internal/lightning"
)

type mockLightningRepo struct {
    invoice   *lightning.Invoice
    saveErr   error
    getErr    error
    updateErr error
}

func (m *mockLightningRepo) GenerateBolt11Invoice(ctx context.Context, taskSlug string, amountSats int64) (*lightning.Invoice, error) {
    return nil, nil
}

func (m *mockLightningRepo) SubscribeInvoiceSettlements(ctx context.Context, callback func(settledInvoice *lightning.Invoice)) error {
    return nil
}

func (m *mockLightningRepo) SaveInvoice(ctx context.Context, invoice *lightning.Invoice) error {
    if m.saveErr != nil {
        return m.saveErr
    }
    m.invoice = invoice
    return nil
}

func (m *mockLightningRepo) GetByPaymentHash(ctx context.Context, paymentHash string) (*lightning.Invoice, error) {
    if m.getErr != nil {
        return nil, m.getErr
    }
    if m.invoice != nil && m.invoice.PaymentHash == paymentHash {
        return m.invoice, nil
    }
    return nil, errors.New("not found")
}

func (m *mockLightningRepo) UpdateSettlement(ctx context.Context, paymentHash string, settledAt time.Time) error {
    if m.updateErr != nil {
        return m.updateErr
    }
    if m.invoice != nil && m.invoice.PaymentHash == paymentHash {
        m.invoice.Settled = true
        m.invoice.SettledAt = settledAt
        return nil
    }
    return errors.New("not found")
}

func TestRequestDonationInvoice(t *testing.T) {
    repo := &mockLightningRepo{}
    svc := NewLightningService(repo, &config.Config{}, nil)

    invoice, err := svc.RequestDonationInvoice(context.Background(), "task1", 100)
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if invoice == nil || invoice.AmountSats != 100 {
        t.Fatalf("expected invoice returned, got %#v", invoice)
    }
}

func TestProcessIncomingSettlement(t *testing.T) {
    repo := &mockLightningRepo{invoice: &lightning.Invoice{PaymentHash: "hash1", TaskSlug: "task1", AmountSats: 100}}
    svc := NewLightningService(repo, &config.Config{}, nil)

    err := svc.ProcessIncomingSettlement(context.Background(), repo.invoice)
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if !repo.invoice.Settled {
        t.Fatal("expected invoice to be marked settled")
    }
}

func TestProcessIncomingSettlementAlreadySettled(t *testing.T) {
    repo := &mockLightningRepo{invoice: &lightning.Invoice{PaymentHash: "hash1", TaskSlug: "task1", AmountSats: 100, Settled: true}}
    svc := NewLightningService(repo, &config.Config{}, nil)

    err := svc.ProcessIncomingSettlement(context.Background(), repo.invoice)
    if err == nil {
        t.Fatal("expected error for already settled invoice")
    }
}

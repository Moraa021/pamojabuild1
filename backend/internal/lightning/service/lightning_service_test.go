package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"pamojabuild1/backend/internal/config"
	"pamojabuild1/backend/internal/events"
	"pamojabuild1/backend/internal/lightning"
)

type mockLightningRepo struct {
	invoice   *lightning.Invoice
	saveErr   error
	getErr    error
	updateErr error
}

type mockLightningNode struct {
	createErr error
	request   lightning.InvoiceRequest
	invoice   *lightning.Invoice
}

const testPaymentHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func (m *mockLightningNode) CreateInvoice(ctx context.Context, request lightning.InvoiceRequest) (*lightning.Invoice, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	m.request = request
	if m.invoice != nil {
		return m.invoice, nil
	}
	now := time.Now().UTC()
	return &lightning.Invoice{
		PaymentRequest: "lnbc100...",
		PaymentHash:    testPaymentHash,
		AmountSats:     request.AmountSats,
		TaskSlug:       request.TaskSlug,
		Status:         lightning.InvoiceStatusPending,
		CreatedAt:      now,
		ExpiresAt:      now.Add(time.Hour),
	}, nil
}

func (m *mockLightningNode) SubscribeInvoiceSettlements(ctx context.Context, sinceSettleIndex int64, handler lightning.SettlementHandler) error {
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

func (m *mockLightningRepo) MarkSettled(ctx context.Context, paymentHash string, settledAt time.Time, settleIndex int64) (bool, error) {
	if m.updateErr != nil {
		return false, m.updateErr
	}
	if m.invoice != nil && m.invoice.PaymentHash == paymentHash {
		if m.invoice.Status != lightning.InvoiceStatusPending {
			return false, nil
		}
		m.invoice.Settled = true
		m.invoice.Status = lightning.InvoiceStatusSettled
		m.invoice.SettledAt = settledAt
		m.invoice.SettleIndex = settleIndex
		return true, nil
	}
	return false, errors.New("not found")
}

func TestRequestDonationInvoice(t *testing.T) {
	repo := &mockLightningRepo{}
	node := &mockLightningNode{}
	svc := NewLightningService(repo, node, &config.Config{}, nil)

	invoice, err := svc.RequestDonationInvoice(context.Background(), "task1", 100)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if invoice == nil || invoice.AmountSats != 100 {
		t.Fatalf("expected invoice returned, got %#v", invoice)
	}
	if invoice.Status != lightning.InvoiceStatusPending {
		t.Fatalf("expected pending status, got %q", invoice.Status)
	}
	if invoice.CreatedAt.IsZero() || invoice.ExpiresAt.IsZero() {
		t.Fatalf("expected invoice timestamps to be set, got %#v", invoice)
	}
	if repo.invoice == nil || repo.invoice.PaymentHash != invoice.PaymentHash {
		t.Fatal("expected generated invoice to be saved")
	}
	if node.request.Memo != "PamojaBuild donation task=task1" {
		t.Fatalf("expected task slug in invoice memo, got %q", node.request.Memo)
	}
}

func TestRequestDonationInvoiceRejectsInvalidNodeInvoice(t *testing.T) {
	repo := &mockLightningRepo{}
	node := &mockLightningNode{invoice: &lightning.Invoice{
		PaymentRequest: "lnbc100...",
		PaymentHash:    "not-a-real-payment-hash",
		AmountSats:     100,
		TaskSlug:       "task1",
		Status:         lightning.InvoiceStatusPending,
	}}
	svc := NewLightningService(repo, node, &config.Config{}, nil)

	_, err := svc.RequestDonationInvoice(context.Background(), "task1", 100)
	if !errors.Is(err, ErrInvalidInvoice) {
		t.Fatalf("expected ErrInvalidInvoice, got %v", err)
	}
	if repo.invoice != nil {
		t.Fatal("expected invalid invoice not to be saved")
	}
}

func TestProcessIncomingSettlement(t *testing.T) {
	repo := &mockLightningRepo{invoice: &lightning.Invoice{PaymentHash: testPaymentHash, TaskSlug: "task1", AmountSats: 100, Status: lightning.InvoiceStatusPending}}
	bus := events.NewEventBus()
	published := 0
	bus.Subscribe(events.PaymentSettled, func(e events.Event) {
		published++
	})
	svc := NewLightningService(repo, &mockLightningNode{}, &config.Config{}, bus)

	err := svc.ProcessIncomingSettlement(context.Background(), repo.invoice)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.invoice.Settled {
		t.Fatal("expected invoice to be marked settled")
	}
	if published != 1 {
		t.Fatalf("expected one settlement event, got %d", published)
	}
}

func TestProcessIncomingSettlementAlreadySettledIsIdempotent(t *testing.T) {
	repo := &mockLightningRepo{invoice: &lightning.Invoice{PaymentHash: testPaymentHash, TaskSlug: "task1", AmountSats: 100, Settled: true, Status: lightning.InvoiceStatusSettled}}
	bus := events.NewEventBus()
	published := 0
	bus.Subscribe(events.PaymentSettled, func(e events.Event) {
		published++
	})
	svc := NewLightningService(repo, &mockLightningNode{}, &config.Config{}, bus)

	err := svc.ProcessIncomingSettlement(context.Background(), repo.invoice)
	if err != nil {
		t.Fatalf("expected no error for duplicate settlement, got %v", err)
	}
	if published != 0 {
		t.Fatalf("expected no duplicate settlement event, got %d", published)
	}
}

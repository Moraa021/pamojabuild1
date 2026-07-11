package service

import (
	"context"
	"database/sql"
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
	expireErr error
	latest    int64
}

type mockLightningNode struct {
	createErr         error
	request           lightning.InvoiceRequest
	invoice           *lightning.Invoice
	subscribeSince    int64
	settlement        *lightning.Invoice
	cancelOnSubscribe context.CancelFunc
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
	m.subscribeSince = sinceSettleIndex
	if m.settlement != nil {
		if err := handler(ctx, m.settlement); err != nil {
			return err
		}
	}
	if m.cancelOnSubscribe != nil {
		m.cancelOnSubscribe()
	}
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
		if m.invoice.Status != lightning.InvoiceStatusPending && m.invoice.Status != lightning.InvoiceStatusExpired {
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

func (m *mockLightningRepo) LatestSettleIndex(ctx context.Context) (int64, error) {
	return m.latest, nil
}

func (m *mockLightningRepo) ExpirePendingInvoices(ctx context.Context, now time.Time) (int64, error) {
	if m.expireErr != nil {
		return 0, m.expireErr
	}
	if m.invoice == nil || m.invoice.Status != lightning.InvoiceStatusPending || m.invoice.ExpiresAt.IsZero() || m.invoice.ExpiresAt.After(now) {
		return 0, nil
	}
	m.invoice.Status = lightning.InvoiceStatusExpired
	return 1, nil
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

func TestGetInvoiceStatusMarksExpiredInvoice(t *testing.T) {
	repo := &mockLightningRepo{invoice: &lightning.Invoice{
		PaymentHash: testPaymentHash,
		TaskSlug:    "task1",
		AmountSats:  100,
		Status:      lightning.InvoiceStatusPending,
		CreatedAt:   time.Now().UTC().Add(-2 * time.Hour),
		ExpiresAt:   time.Now().UTC().Add(-time.Hour),
	}}
	svc := NewLightningService(repo, &mockLightningNode{}, &config.Config{}, nil)

	invoice, err := svc.GetInvoiceStatus(context.Background(), testPaymentHash)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if invoice.Status != lightning.InvoiceStatusExpired {
		t.Fatalf("expected expired invoice status, got %q", invoice.Status)
	}
	if invoice.Settled {
		t.Fatal("expected expired invoice to remain unsettled")
	}
}

func TestGetInvoiceStatusRejectsInvalidPaymentHash(t *testing.T) {
	svc := NewLightningService(&mockLightningRepo{}, &mockLightningNode{}, &config.Config{}, nil)

	_, err := svc.GetInvoiceStatus(context.Background(), "not-a-payment-hash")
	if !errors.Is(err, lightning.ErrInvalidPaymentHash) {
		t.Fatalf("expected ErrInvalidPaymentHash, got %v", err)
	}
}

func TestProcessIncomingSettlementIgnoresUnknownPaymentHash(t *testing.T) {
	repo := &mockLightningRepo{getErr: sql.ErrNoRows}
	bus := events.NewEventBus()
	published := 0
	bus.Subscribe(events.PaymentSettled, func(e events.Event) {
		published++
	})
	svc := NewLightningService(repo, &mockLightningNode{}, &config.Config{}, bus)

	err := svc.ProcessIncomingSettlement(context.Background(), &lightning.Invoice{PaymentHash: testPaymentHash})
	if err != nil {
		t.Fatalf("expected unknown settlement to be ignored, got %v", err)
	}
	if published != 0 {
		t.Fatalf("expected no event for unknown settlement, got %d", published)
	}
}

func TestStartSettlementListenerSubscribesFromLatestIndex(t *testing.T) {
	repo := &mockLightningRepo{
		invoice: &lightning.Invoice{
			PaymentHash: testPaymentHash,
			TaskSlug:    "task1",
			AmountSats:  100,
			Status:      lightning.InvoiceStatusPending,
		},
		latest: 12,
	}
	ctx, cancel := context.WithCancel(context.Background())
	node := &mockLightningNode{
		settlement: &lightning.Invoice{
			PaymentHash: testPaymentHash,
			SettledAt:   time.Now().UTC(),
			SettleIndex: 13,
		},
		cancelOnSubscribe: cancel,
	}
	bus := events.NewEventBus()
	published := 0
	bus.Subscribe(events.PaymentSettled, func(e events.Event) {
		published++
	})
	svc := NewLightningService(repo, node, &config.Config{}, bus)

	svc.StartSettlementListener(ctx)

	if node.subscribeSince != 12 {
		t.Fatalf("expected listener to resume from settle index 12, got %d", node.subscribeSince)
	}
	if repo.invoice.Status != lightning.InvoiceStatusSettled {
		t.Fatalf("expected invoice to be settled, got status %q", repo.invoice.Status)
	}
	if repo.invoice.SettleIndex != 13 {
		t.Fatalf("expected settle index 13, got %d", repo.invoice.SettleIndex)
	}
	if published != 1 {
		t.Fatalf("expected one settlement event, got %d", published)
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

func TestProcessIncomingSettlementCreditsExpiredInvoiceWhenLNDSettlesIt(t *testing.T) {
	repo := &mockLightningRepo{invoice: &lightning.Invoice{PaymentHash: testPaymentHash, TaskSlug: "task1", AmountSats: 100, Status: lightning.InvoiceStatusExpired}}
	bus := events.NewEventBus()
	published := 0
	bus.Subscribe(events.PaymentSettled, func(e events.Event) {
		published++
	})
	svc := NewLightningService(repo, &mockLightningNode{}, &config.Config{}, bus)

	err := svc.ProcessIncomingSettlement(context.Background(), &lightning.Invoice{PaymentHash: testPaymentHash})
	if err != nil {
		t.Fatalf("expected expired invoice settlement to be credited, got %v", err)
	}
	if repo.invoice.Status != lightning.InvoiceStatusSettled {
		t.Fatalf("expected invoice to be settled, got %q", repo.invoice.Status)
	}
	if published != 1 {
		t.Fatalf("expected one settlement event for expired invoice, got %d", published)
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

package service

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"pamojabuild1/backend/internal/config"
	"pamojabuild1/backend/internal/events"
	"pamojabuild1/backend/internal/lightning"
)

var (
	ErrInvoiceGeneration = errors.New("failed to generate invoice")
	ErrInvalidInvoice    = errors.New("generated invoice is missing required fields")
)

const defaultInvoiceExpiry = time.Hour
const paymentHashHexLength = 64
const settlementListenerRetryDelay = 5 * time.Second
const invoiceExpirySweepInterval = time.Minute

type LightningService struct {
	repo     lightning.Repository
	node     lightning.NodeClient
	cfg      *config.Config
	eventBus *events.EventBus
}

func NewLightningService(repo lightning.Repository, node lightning.NodeClient, cfg *config.Config, eventBus *events.EventBus) *LightningService {
	return &LightningService{repo: repo, node: node, cfg: cfg, eventBus: eventBus}
}

func (s *LightningService) RequestDonationInvoice(ctx context.Context, taskSlug string, amountSats int64) (*lightning.Invoice, error) {
	if amountSats <= 0 {
		return nil, errors.New("amount must be greater than 0")
	}
	if s.node == nil {
		return nil, ErrInvoiceGeneration
	}

	invoice, err := s.node.CreateInvoice(ctx, lightning.InvoiceRequest{
		TaskSlug:   taskSlug,
		AmountSats: amountSats,
		Memo:       donationInvoiceMemo(taskSlug),
		Expiry:     defaultInvoiceExpiry,
	})
	if err != nil {
		return nil, ErrInvoiceGeneration
	}
	if invoice == nil || invoice.PaymentRequest == "" || !isPaymentHashHex(invoice.PaymentHash) {
		return nil, ErrInvalidInvoice
	}

	now := time.Now().UTC()
	if invoice.TaskSlug == "" {
		invoice.TaskSlug = taskSlug
	}
	if invoice.AmountSats == 0 {
		invoice.AmountSats = amountSats
	}
	if invoice.Status == "" {
		invoice.Status = lightning.InvoiceStatusPending
	}
	if invoice.CreatedAt.IsZero() {
		invoice.CreatedAt = now
	}
	if invoice.ExpiresAt.IsZero() {
		invoice.ExpiresAt = invoice.CreatedAt.Add(defaultInvoiceExpiry)
	}

	if err := s.repo.SaveInvoice(ctx, invoice); err != nil {
		return nil, ErrInvoiceGeneration
	}

	return invoice, nil
}

func donationInvoiceMemo(taskSlug string) string {
	// The memo gives operators and wallets human-readable context, but accounting
	// attribution still relies on our payment_hash -> task_slug database record.
	return fmt.Sprintf("PamojaBuild donation task=%s", taskSlug)
}

func isPaymentHashHex(paymentHash string) bool {
	if len(paymentHash) != paymentHashHexLength {
		return false
	}
	_, err := hex.DecodeString(strings.ToLower(paymentHash))
	return err == nil
}

func (s *LightningService) GetInvoiceStatus(ctx context.Context, paymentHash string) (*lightning.Invoice, error) {
	if !isPaymentHashHex(paymentHash) {
		return nil, lightning.ErrInvalidPaymentHash
	}
	if _, err := s.repo.ExpirePendingInvoices(ctx, time.Now().UTC()); err != nil {
		return nil, err
	}

	invoice, err := s.repo.GetByPaymentHash(ctx, paymentHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, lightning.ErrInvoiceNotFound
		}
		return nil, err
	}
	return invoice, nil
}

func (s *LightningService) ProcessIncomingSettlement(ctx context.Context, invoice *lightning.Invoice) error {
	if invoice == nil || invoice.PaymentHash == "" {
		return errors.New("settlement invoice must include payment hash")
	}

	existing, err := s.repo.GetByPaymentHash(ctx, invoice.PaymentHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return s.advanceSettlementCursor(ctx, invoice)
		}
		return err
	}

	settledAt := invoice.SettledAt
	if settledAt.IsZero() {
		settledAt = time.Now().UTC()
	}

	changed, err := s.repo.MarkSettled(ctx, invoice.PaymentHash, settledAt, invoice.SettleIndex)
	if err != nil {
		return err
	}
	if !changed {
		return s.advanceSettlementCursor(ctx, invoice)
	}

	if s.eventBus != nil {
		s.eventBus.Publish(events.Event{
			Type: events.PaymentSettled,
			Payload: events.PaymentSettledPayload{
				TaskSlug:    existing.TaskSlug,
				AmountSats:  existing.AmountSats,
				PaymentHash: existing.PaymentHash,
			},
		})
	}

	return s.advanceSettlementCursor(ctx, invoice)
}

func (s *LightningService) advanceSettlementCursor(ctx context.Context, invoice *lightning.Invoice) error {
	if invoice == nil || invoice.SettleIndex <= 0 {
		return nil
	}
	return s.repo.AdvanceSettlementCursor(ctx, invoice.SettleIndex)
}

func (s *LightningService) StartSettlementListener(ctx context.Context) {
	if s.node == nil {
		log.Println("lightning settlement listener not started: node client is nil")
		return
	}
	if s.repo == nil {
		log.Println("lightning settlement listener not started: repository is nil")
		return
	}

	for {
		if err := ctx.Err(); err != nil {
			return
		}

		// LND settlement indexes are monotonic. Reading the latest persisted
		// value before each subscription lets the listener resume after a
		// process restart or stream reconnect without replaying older invoices.
		sinceSettleIndex, err := s.repo.LatestSettleIndex(ctx)
		if err != nil {
			log.Printf("lightning settlement listener could not load resume index: %v", err)
			if !waitForSettlementListenerRetry(ctx) {
				return
			}
			continue
		}

		err = s.node.SubscribeInvoiceSettlements(ctx, sinceSettleIndex, s.ProcessIncomingSettlement)
		if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			if ctx.Err() != nil {
				return
			}
			log.Println("lightning settlement listener ended; retrying subscription")
		} else {
			log.Printf("lightning settlement listener error: %v", err)
		}

		if !waitForSettlementListenerRetry(ctx) {
			return
		}
	}
}

func waitForSettlementListenerRetry(ctx context.Context) bool {
	timer := time.NewTimer(settlementListenerRetryDelay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (s *LightningService) StartInvoiceExpiryWorker(ctx context.Context) {
	if s.repo == nil {
		log.Println("lightning invoice expiry worker not started: repository is nil")
		return
	}

	s.expirePendingInvoices(ctx)

	ticker := time.NewTicker(invoiceExpirySweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.expirePendingInvoices(ctx)
		}
	}
}

func (s *LightningService) expirePendingInvoices(ctx context.Context) {
	expired, err := s.repo.ExpirePendingInvoices(ctx, time.Now().UTC())
	if err != nil {
		log.Printf("lightning invoice expiry worker error: %v", err)
		return
	}
	if expired > 0 {
		log.Printf("lightning invoice expiry worker marked %d invoice(s) expired", expired)
	}
}

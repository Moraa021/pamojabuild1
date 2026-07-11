package service

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
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

func (s *LightningService) ProcessIncomingSettlement(ctx context.Context, invoice *lightning.Invoice) error {
	if invoice == nil || invoice.PaymentHash == "" {
		return errors.New("settlement invoice must include payment hash")
	}

	existing, err := s.repo.GetByPaymentHash(ctx, invoice.PaymentHash)
	if err != nil {
		return err
	}

	if existing.Status == lightning.InvoiceStatusExpired {
		return errors.New("cannot settle expired invoice")
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
		return nil
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

	return nil
}

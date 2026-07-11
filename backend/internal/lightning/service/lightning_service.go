package service

import (
	"context"
	"errors"
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

type LightningService struct {
	repo     lightning.Client
	cfg      *config.Config
	eventBus *events.EventBus
}

func NewLightningService(repo lightning.Client, cfg *config.Config, eventBus *events.EventBus) *LightningService {
	return &LightningService{repo: repo, cfg: cfg, eventBus: eventBus}
}

func (s *LightningService) RequestDonationInvoice(ctx context.Context, taskSlug string, amountSats int64) (*lightning.Invoice, error) {
	if amountSats <= 0 {
		return nil, errors.New("amount must be greater than 0")
	}

	invoice, err := s.repo.GenerateBolt11Invoice(ctx, taskSlug, amountSats)
	if err != nil {
		return nil, ErrInvoiceGeneration
	}
	if invoice == nil || invoice.PaymentRequest == "" || invoice.PaymentHash == "" {
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

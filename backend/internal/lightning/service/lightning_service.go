package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"pamojabuild1/backend/internal/config"
	"pamojabuild1/backend/internal/events"
	"pamojabuild1/backend/internal/lightning"
)

var (
	ErrInvoiceGeneration = errors.New("failed to generate invoice")
)

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

	// In production, this calls LND to generate a real BOLT11 invoice
	invoice := &lightning.Invoice{
		PaymentRequest: fmt.Sprintf("lnbc%d...", amountSats), // Placeholder
		PaymentHash:    fmt.Sprintf("hash_%s_%d", taskSlug, time.Now().Unix()),
		AmountSats:     amountSats,
		TaskSlug:       taskSlug,
		Settled:        false,
	}

	if err := s.repo.SaveInvoice(ctx, invoice); err != nil {
		return nil, ErrInvoiceGeneration
	}

	return invoice, nil
}

func (s *LightningService) ProcessIncomingSettlement(ctx context.Context, invoice *lightning.Invoice) error {
	existing, err := s.repo.GetByPaymentHash(ctx, invoice.PaymentHash)
	if err != nil {
		return err
	}

	if existing.Settled {
		return errors.New("invoice already settled")
	}

	invoice.Settled = true
	invoice.SettledAt = time.Now()

	if err := s.repo.UpdateSettlement(ctx, invoice.PaymentHash, invoice.SettledAt); err != nil {
		return err
	}

	if s.eventBus != nil {
		s.eventBus.Publish(events.Event{
			Type: events.PaymentSettled,
			Payload: events.PaymentSettledPayload{
				TaskSlug:    invoice.TaskSlug,
				AmountSats:  invoice.AmountSats,
				PaymentHash: invoice.PaymentHash,
			},
		})
	}

	return nil
}
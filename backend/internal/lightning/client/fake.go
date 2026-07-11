package client

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"pamojabuild1/backend/internal/lightning"
)

type FakeNodeClient struct {
	mu       sync.Mutex
	invoices []*lightning.Invoice

	CreateInvoiceErr error
}

func NewFakeNodeClient() *FakeNodeClient {
	return &FakeNodeClient{}
}

func (c *FakeNodeClient) CreateInvoice(ctx context.Context, request lightning.InvoiceRequest) (*lightning.Invoice, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c.CreateInvoiceErr != nil {
		return nil, c.CreateInvoiceErr
	}
	if request.TaskSlug == "" {
		return nil, errors.New("task slug is required")
	}
	if request.AmountSats <= 0 {
		return nil, errors.New("amount must be greater than 0")
	}

	paymentHash, err := randomHex(32)
	if err != nil {
		return nil, fmt.Errorf("generate fake payment hash: %w", err)
	}

	now := time.Now().UTC()
	expiry := request.Expiry
	if expiry <= 0 {
		expiry = time.Hour
	}

	invoice := &lightning.Invoice{
		PaymentRequest: fmt.Sprintf("lnbc%d-%s", request.AmountSats, paymentHash),
		PaymentHash:    paymentHash,
		AmountSats:     request.AmountSats,
		TaskSlug:       request.TaskSlug,
		Status:         lightning.InvoiceStatusPending,
		CreatedAt:      now,
		ExpiresAt:      now.Add(expiry),
	}

	c.mu.Lock()
	c.invoices = append(c.invoices, invoice)
	c.mu.Unlock()

	return invoice, nil
}

func (c *FakeNodeClient) SubscribeInvoiceSettlements(ctx context.Context, sinceSettleIndex int64, handler lightning.SettlementHandler) error {
	if handler == nil {
		return errors.New("settlement handler is required")
	}
	<-ctx.Done()
	return ctx.Err()
}

func randomHex(byteCount int) (string, error) {
	bytes := make([]byte, byteCount)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

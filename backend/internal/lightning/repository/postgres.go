package repository

import (
	"context"
	"database/sql"

	"pamojabuild1/backend/internal/lightning"
)

type LightningRepository struct {
	db *sql.DB
}

func NewLightningRepository(db *sql.DB) *LightningRepository {
	return &LightningRepository{db: db}
}

func (r *LightningRepository) SaveInvoice(ctx context.Context, invoice *lightning.Invoice) error {
	query := `
		INSERT INTO lightning_invoices (payment_request, payment_hash, amount_sats, task_slug, settled, settled_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.db.ExecContext(ctx, query,
		invoice.PaymentRequest, invoice.PaymentHash, invoice.AmountSats,
		invoice.TaskSlug, invoice.Settled, invoice.SettledAt,
	)
	return err
}

func (r *LightningRepository) GetByPaymentHash(ctx context.Context, paymentHash string) (*lightning.Invoice, error) {
	invoice := &lightning.Invoice{}
	query := `
		SELECT payment_request, payment_hash, amount_sats, task_slug, settled, settled_at
		FROM lightning_invoices WHERE payment_hash = $1`

	err := r.db.QueryRowContext(ctx, query, paymentHash).Scan(
		&invoice.PaymentRequest, &invoice.PaymentHash, &invoice.AmountSats,
		&invoice.TaskSlug, &invoice.Settled, &invoice.SettledAt,
	)
	if err != nil {
		return nil, err
	}
	return invoice, nil
}

func (r *LightningRepository) UpdateSettlement(ctx context.Context, paymentHash string, settledAt interface{}) error {
	query := `UPDATE lightning_invoices SET settled = true, settled_at = $1 WHERE payment_hash = $2`
	_, err := r.db.ExecContext(ctx, query, settledAt, paymentHash)
	return err
}
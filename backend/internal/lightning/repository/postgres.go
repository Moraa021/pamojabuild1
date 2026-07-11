package repository

import (
	"context"
	"database/sql"
	"time"

	"pamojabuild1/backend/internal/lightning"
)

type LightningRepository struct {
	db *sql.DB
}

func NewLightningRepository(db *sql.DB) *LightningRepository {
	return &LightningRepository{db: db}
}

func (r *LightningRepository) SaveInvoice(ctx context.Context, invoice *lightning.Invoice) error {
	if invoice.Status == "" {
		invoice.Status = lightning.InvoiceStatusPending
	}

	query := `
		INSERT INTO lightning_invoices (
			payment_request, payment_hash, amount_sats, task_slug, status,
			settled, created_at, expires_at, settled_at, add_index, settle_index
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := r.db.ExecContext(ctx, query,
		invoice.PaymentRequest, invoice.PaymentHash, invoice.AmountSats, invoice.TaskSlug,
		invoice.Status, invoice.Settled, invoice.CreatedAt, invoice.ExpiresAt,
		nullableTime(invoice.SettledAt), invoice.AddIndex, invoice.SettleIndex,
	)
	return err
}

func (r *LightningRepository) GetByPaymentHash(ctx context.Context, paymentHash string) (*lightning.Invoice, error) {
	invoice := &lightning.Invoice{}
	var settledAt sql.NullTime
	var expiresAt sql.NullTime
	query := `
		SELECT
			payment_request, payment_hash, amount_sats, task_slug, status,
			settled, created_at, expires_at, settled_at, add_index, settle_index
		FROM lightning_invoices WHERE payment_hash = $1`

	err := r.db.QueryRowContext(ctx, query, paymentHash).Scan(
		&invoice.PaymentRequest, &invoice.PaymentHash, &invoice.AmountSats,
		&invoice.TaskSlug, &invoice.Status, &invoice.Settled, &invoice.CreatedAt,
		&expiresAt, &settledAt, &invoice.AddIndex, &invoice.SettleIndex,
	)
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		invoice.ExpiresAt = expiresAt.Time
	}
	if settledAt.Valid {
		invoice.SettledAt = settledAt.Time
	}
	return invoice, nil
}

func (r *LightningRepository) MarkSettled(ctx context.Context, paymentHash string, settledAt time.Time, settleIndex int64) (bool, error) {
	query := `
		UPDATE lightning_invoices
		SET settled = true, status = $1, settled_at = $2, settle_index = $3
		WHERE payment_hash = $4 AND status = $5`
	result, err := r.db.ExecContext(ctx, query,
		lightning.InvoiceStatusSettled,
		settledAt,
		settleIndex,
		paymentHash,
		lightning.InvoiceStatusPending,
	)
	if err != nil {
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected == 1, nil
}

func nullableTime(value time.Time) interface{} {
	if value.IsZero() {
		return nil
	}
	return value
}

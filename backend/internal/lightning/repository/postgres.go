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

const settlementCursorKey = "lnd_settle_index"

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
	// Expired invoices are still eligible here because LND settlement is the
	// final accounting signal. Expiry means "unpaid past the wallet deadline";
	// if LND later confirms funds for a known invoice, we must credit them once.
	query := `
		UPDATE lightning_invoices
		SET settled = true, status = $1, settled_at = $2, settle_index = $3
		WHERE payment_hash = $4 AND status IN ($5, $6)`
	result, err := r.db.ExecContext(ctx, query,
		lightning.InvoiceStatusSettled,
		settledAt,
		settleIndex,
		paymentHash,
		lightning.InvoiceStatusPending,
		lightning.InvoiceStatusExpired,
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

func (r *LightningRepository) LatestSettleIndex(ctx context.Context) (int64, error) {
	var latestInvoiceIndex sql.NullInt64
	query := `SELECT MAX(settle_index) FROM lightning_invoices WHERE status = $1`
	if err := r.db.QueryRowContext(ctx, query, lightning.InvoiceStatusSettled).Scan(&latestInvoiceIndex); err != nil {
		return 0, err
	}

	latestIndex := int64(0)
	if latestInvoiceIndex.Valid {
		latestIndex = latestInvoiceIndex.Int64
	}

	var cursorIndex int64
	cursorQuery := `SELECT value_integer FROM lightning_sync_state WHERE key = $1`
	err := r.db.QueryRowContext(ctx, cursorQuery, settlementCursorKey).Scan(&cursorIndex)
	if err != nil {
		if err == sql.ErrNoRows {
			return latestIndex, nil
		}
		return 0, err
	}
	if cursorIndex > latestIndex {
		return cursorIndex, nil
	}
	return latestIndex, nil
}

func (r *LightningRepository) AdvanceSettlementCursor(ctx context.Context, settleIndex int64) error {
	if settleIndex <= 0 {
		return nil
	}

	// The cursor tracks progress through LND's global invoice stream, including
	// settled invoices that do not belong to this app. It only moves forward so
	// old replayed updates cannot hide newer work after restart.
	query := `
		INSERT INTO lightning_sync_state (key, value_integer, updated_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET
			value_integer = CASE
				WHEN lightning_sync_state.value_integer < EXCLUDED.value_integer
					THEN EXCLUDED.value_integer
				ELSE lightning_sync_state.value_integer
			END,
			updated_at = CURRENT_TIMESTAMP`
	_, err := r.db.ExecContext(ctx, query, settlementCursorKey, settleIndex)
	return err
}

func (r *LightningRepository) ExpirePendingInvoices(ctx context.Context, now time.Time) (int64, error) {
	query := `
		UPDATE lightning_invoices
		SET status = $1
		WHERE status = $2 AND expires_at IS NOT NULL AND expires_at <= $3`
	result, err := r.db.ExecContext(ctx, query,
		lightning.InvoiceStatusExpired,
		lightning.InvoiceStatusPending,
		now,
	)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return rowsAffected, nil
}

func nullableTime(value time.Time) interface{} {
	if value.IsZero() {
		return nil
	}
	return value
}

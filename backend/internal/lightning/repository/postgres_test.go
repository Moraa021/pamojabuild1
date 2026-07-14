package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"pamojabuild1/backend/internal/lightning"
)

func newTestLightningRepository(t *testing.T) *LightningRepository {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite database: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	schema := `
		CREATE TABLE lightning_invoices (
			payment_request TEXT NOT NULL,
			payment_hash VARCHAR(255) PRIMARY KEY,
			amount_sats INTEGER NOT NULL,
			task_slug VARCHAR(255),
			settled INTEGER DEFAULT 0,
			settled_at TIMESTAMP,
			status VARCHAR(32) NOT NULL DEFAULT 'pending',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP,
			add_index INTEGER NOT NULL DEFAULT 0,
			settle_index INTEGER NOT NULL DEFAULT 0
		);

		CREATE TABLE lightning_sync_state (
			key VARCHAR(128) PRIMARY KEY,
			value_integer INTEGER NOT NULL,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to create lightning test schema: %v", err)
	}

	return NewLightningRepository(db)
}

func TestLightningRepositorySettlementCursorOnlyMovesForward(t *testing.T) {
	repo := newTestLightningRepository(t)
	ctx := context.Background()

	if err := repo.AdvanceSettlementCursor(ctx, 15); err != nil {
		t.Fatalf("expected cursor advance, got %v", err)
	}
	if err := repo.AdvanceSettlementCursor(ctx, 9); err != nil {
		t.Fatalf("expected stale cursor advance to be harmless, got %v", err)
	}

	latest, err := repo.LatestSettleIndex(ctx)
	if err != nil {
		t.Fatalf("expected latest settle index, got %v", err)
	}
	if latest != 15 {
		t.Fatalf("expected cursor to remain at 15, got %d", latest)
	}
}

func TestLightningRepositoryLatestSettleIndexUsesSettledInvoicesWhenAhead(t *testing.T) {
	repo := newTestLightningRepository(t)
	ctx := context.Background()

	if err := repo.AdvanceSettlementCursor(ctx, 7); err != nil {
		t.Fatalf("expected cursor advance, got %v", err)
	}
	if err := repo.SaveInvoice(ctx, &lightning.Invoice{
		PaymentRequest: "lnbc100...",
		PaymentHash:    "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		AmountSats:     100,
		TaskSlug:       "task1",
		Status:         lightning.InvoiceStatusSettled,
		Settled:        true,
		CreatedAt:      time.Now().UTC(),
		SettledAt:      time.Now().UTC(),
		SettleIndex:    21,
	}); err != nil {
		t.Fatalf("expected invoice save, got %v", err)
	}

	latest, err := repo.LatestSettleIndex(ctx)
	if err != nil {
		t.Fatalf("expected latest settle index, got %v", err)
	}
	if latest != 21 {
		t.Fatalf("expected settled invoice index 21, got %d", latest)
	}
}

func TestLightningRepositoryExpiresOnlyPendingInvoices(t *testing.T) {
	repo := newTestLightningRepository(t)
	ctx := context.Background()
	now := time.Now().UTC()

	if err := repo.SaveInvoice(ctx, &lightning.Invoice{
		PaymentRequest: "lnbc100...",
		PaymentHash:    "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		AmountSats:     100,
		TaskSlug:       "task1",
		Status:         lightning.InvoiceStatusPending,
		CreatedAt:      now.Add(-2 * time.Hour),
		ExpiresAt:      now.Add(-time.Hour),
	}); err != nil {
		t.Fatalf("expected pending invoice save, got %v", err)
	}
	if err := repo.SaveInvoice(ctx, &lightning.Invoice{
		PaymentRequest: "lnbc200...",
		PaymentHash:    "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		AmountSats:     200,
		TaskSlug:       "task1",
		Status:         lightning.InvoiceStatusSettled,
		Settled:        true,
		CreatedAt:      now.Add(-2 * time.Hour),
		ExpiresAt:      now.Add(-time.Hour),
		SettledAt:      now.Add(-90 * time.Minute),
		SettleIndex:    3,
	}); err != nil {
		t.Fatalf("expected settled invoice save, got %v", err)
	}

	expired, err := repo.ExpirePendingInvoices(ctx, now)
	if err != nil {
		t.Fatalf("expected expiry sweep, got %v", err)
	}
	if expired != 1 {
		t.Fatalf("expected one invoice to expire, got %d", expired)
	}

	pendingInvoice, err := repo.GetByPaymentHash(ctx, "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")
	if err != nil {
		t.Fatalf("expected expired invoice lookup, got %v", err)
	}
	if pendingInvoice.Status != lightning.InvoiceStatusExpired {
		t.Fatalf("expected pending invoice to become expired, got %q", pendingInvoice.Status)
	}

	settledInvoice, err := repo.GetByPaymentHash(ctx, "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd")
	if err != nil {
		t.Fatalf("expected settled invoice lookup, got %v", err)
	}
	if settledInvoice.Status != lightning.InvoiceStatusSettled {
		t.Fatalf("expected settled invoice to remain settled, got %q", settledInvoice.Status)
	}
}

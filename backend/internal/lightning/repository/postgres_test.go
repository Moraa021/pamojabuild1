package repository

import (
	"context"
	"testing"
	"time"

	"pamojabuild1/backend/internal/lightning"
	"pamojabuild1/backend/internal/testsupport"
)

func newTestLightningRepository(t *testing.T) *LightningRepository {
	t.Helper()

	database := testsupport.NewPostgresDatabase(t)
	var userID int64
	if err := database.QueryRow(`
		INSERT INTO users (phone_number, password_hash, display_name)
		VALUES ('+254700000099', 'test-hash', 'Lightning Repository')
		RETURNING id`).Scan(&userID); err != nil {
		t.Fatalf("create Lightning repository test user: %v", err)
	}
	if _, err := database.Exec(`
		INSERT INTO tasks (slug, creator_id, title)
		VALUES ('task1', $1, 'Lightning Repository Task')`, userID); err != nil {
		t.Fatalf("create Lightning repository test task: %v", err)
	}

	return NewLightningRepository(database)
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

package repository

import (
	"context"
	"database/sql"

	"pamojabuild1/backend/internal/ledger"
)

type LedgerRepository struct {
	db *sql.DB
}

func NewLedgerRepository(db *sql.DB) *LedgerRepository {
	return &LedgerRepository{db: db}
}

func (r *LedgerRepository) GetLastEntry(ctx context.Context, taskSlug string) (*ledger.LedgerEntry, error) {
	entry := &ledger.LedgerEntry{}
	query := `
		SELECT id, task_slug, entry_type, amount_sats, reference_id, previous_hash, row_hmac
		FROM ledger_entries WHERE task_slug = $1
		ORDER BY id DESC LIMIT 1`

	err := r.db.QueryRowContext(ctx, query, taskSlug).Scan(
		&entry.ID, &entry.TaskSlug, &entry.EntryType, &entry.AmountSats,
		&entry.ReferenceID, &entry.PreviousHash, &entry.RowHMAC,
	)
	if err != nil {
		return nil, err
	}
	return entry, nil
}

func (r *LedgerRepository) AppendEntry(ctx context.Context, entry *ledger.LedgerEntry) error {
	query := `
		INSERT INTO ledger_entries (task_slug, entry_type, amount_sats, reference_id, previous_hash, row_hmac)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`

	return r.db.QueryRowContext(ctx, query,
		entry.TaskSlug, entry.EntryType, entry.AmountSats, entry.ReferenceID,
		entry.PreviousHash, entry.RowHMAC,
	).Scan(&entry.ID)
}

func (r *LedgerRepository) GetTaskBalance(ctx context.Context, taskSlug string) (*ledger.BalanceSummary, error) {
	summary := &ledger.BalanceSummary{}
	query := `
		SELECT 
			COALESCE(SUM(CASE WHEN entry_type = 'INBOUND_DONATION' THEN amount_sats ELSE 0 END), 0) as l2_balance,
			COALESCE(SUM(CASE WHEN entry_type = 'SUBMARINE_SWAP' THEN amount_sats ELSE 0 END), 0) as l1_balance,
			COALESCE(MAX(id), 0) as current_index
		FROM ledger_entries WHERE task_slug = $1`

	err := r.db.QueryRowContext(ctx, query, taskSlug).Scan(
		&summary.L2BalanceSats, &summary.L1BalanceSats, &summary.CurrentIndex,
	)
	if err != nil {
		return nil, err
	}
	return summary, nil
}

func (r *LedgerRepository) GetAllEntries(ctx context.Context, taskSlug string) ([]ledger.LedgerEntry, error) {
	query := `
		SELECT id, task_slug, entry_type, amount_sats, reference_id, previous_hash, row_hmac
		FROM ledger_entries WHERE task_slug = $1
		ORDER BY id ASC`

	rows, err := r.db.QueryContext(ctx, query, taskSlug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []ledger.LedgerEntry
	for rows.Next() {
		var entry ledger.LedgerEntry
		if err := rows.Scan(&entry.ID, &entry.TaskSlug, &entry.EntryType,
			&entry.AmountSats, &entry.ReferenceID, &entry.PreviousHash, &entry.RowHMAC); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (r *LedgerRepository) UpdateBalances(ctx context.Context, taskSlug string, l2Delta, l1Delta int64) error {
	// Placeholder implementation for ledger balance updates.
	return nil
}

func (r *LedgerRepository) IncrementDerivationIndex(ctx context.Context, taskSlug string) error {
	// Placeholder implementation for derivation index management.
	return nil
}

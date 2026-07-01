package service

import (
    "context"
    "testing"

    "pamojabuild1/backend/internal/ledger"
)

type mockLedgerRepo struct {
    entries []*ledger.LedgerEntry
    balance *ledger.BalanceSummary
}

func (m *mockLedgerRepo) GetLastEntry(ctx context.Context, taskSlug string) (*ledger.LedgerEntry, error) {
    if len(m.entries) == 0 {
        return nil, nil
    }
    return m.entries[len(m.entries)-1], nil
}

func (m *mockLedgerRepo) AppendEntry(ctx context.Context, entry *ledger.LedgerEntry) error {
    m.entries = append(m.entries, entry)
    return nil
}

func (m *mockLedgerRepo) GetAllEntries(ctx context.Context, taskSlug string) ([]ledger.LedgerEntry, error) {
    result := make([]ledger.LedgerEntry, len(m.entries))
    for i, entry := range m.entries {
        result[i] = *entry
    }
    return result, nil
}

func (m *mockLedgerRepo) GetTaskBalance(ctx context.Context, taskSlug string) (*ledger.BalanceSummary, error) {
    if m.balance == nil {
        return &ledger.BalanceSummary{}, nil
    }
    return m.balance, nil
}

func (m *mockLedgerRepo) UpdateBalances(ctx context.Context, taskSlug string, l2Delta, l1Delta int64) error {
    return nil
}

func (m *mockLedgerRepo) IncrementDerivationIndex(ctx context.Context, taskSlug string) error {
    return nil
}

func TestRecordValidatedTransaction(t *testing.T) {
    repo := &mockLedgerRepo{}
    svc := NewLedgerService(repo, "secret", nil)

    err := svc.RecordValidatedTransaction(context.Background(), "task1", "deposit", 100, "ref123")
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if len(repo.entries) != 1 {
        t.Fatalf("expected 1 ledger entry, got %d", len(repo.entries))
    }
    if len(repo.entries[0].RowHMAC) == 0 {
        t.Fatal("expected row HMAC to be calculated")
    }
}

func TestVerifyEntireChainIntegrity(t *testing.T) {
    repo := &mockLedgerRepo{}
    svc := NewLedgerService(repo, "secret", nil)

    err := svc.RecordValidatedTransaction(context.Background(), "task1", "deposit", 100, "ref123")
    if err != nil {
        t.Fatalf("setup failed: %v", err)
    }

    ok, err := svc.VerifyEntireChainIntegrity(context.Background(), "task1")
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if !ok {
        t.Fatal("expected ledger integrity to be valid")
    }
}

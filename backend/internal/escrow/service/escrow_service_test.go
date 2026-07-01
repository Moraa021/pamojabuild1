package service

import (
    "context"
    "testing"

    "pamojabuild1/backend/internal/escrow"
    "pamojabuild1/backend/internal/events"
)

type mockSignatureRepo struct {
    saved    *escrow.SignatureCollection
    count    int
    saveErr  error
    countErr error
}

func (m *mockSignatureRepo) SaveSignature(ctx context.Context, sig *escrow.SignatureCollection) error {
    if m.saveErr != nil {
        return m.saveErr
    }
    m.saved = sig
    return nil
}

func (m *mockSignatureRepo) GetSignatures(ctx context.Context, taskSlug string) ([]escrow.SignatureCollection, error) {
    return nil, nil
}

func (m *mockSignatureRepo) GetSignatureCount(ctx context.Context, taskSlug string) (int, error) {
    if m.countErr != nil {
        return 0, m.countErr
    }
    return m.count, nil
}

func TestSubmitTrusteeSignaturePublishesThresholdReached(t *testing.T) {
    repo := &mockSignatureRepo{count: 3}
    bus := events.NewEventBus()
    got := false
    unsub := bus.Subscribe(events.ThresholdReached, func(e events.Event) {
        got = true
    })
    defer unsub()

    svc := NewEscrowService(repo, nil, nil, bus)
    _, err := svc.SubmitTrusteeSignature(context.Background(), "task1", &escrow.SignatureCollection{})
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    bus.Replay()
    if !got {
        t.Fatal("expected ThresholdReached event to be published")
    }
}

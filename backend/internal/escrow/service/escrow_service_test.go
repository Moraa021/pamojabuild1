package service

import (
	"context"
	"database/sql"
	"testing"

	"pamojabuild1/backend/internal/escrow"
	"pamojabuild1/backend/internal/events"
	"pamojabuild1/backend/internal/trustee"
)

type mockSignatureRepo struct {
	saved    *escrow.SignatureCollection
	count    int
	saveErr  error
	countErr error
}

type mockTrusteeRepo struct {
	keys []trustee.TrusteeKey
}

func (m *mockTrusteeRepo) SaveKeys(context.Context, *trustee.TrusteeKey) error {
	return nil
}

func (m *mockTrusteeRepo) GetKeysByTask(context.Context, string) ([]trustee.TrusteeKey, error) {
	return m.keys, nil
}

func (m *mockTrusteeRepo) GetSpecificTrustee(context.Context, string, int32) (*trustee.TrusteeKey, error) {
	return nil, sql.ErrNoRows
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

	trusteeRepo := &mockTrusteeRepo{keys: []trustee.TrusteeKey{{
		UserID:             7,
		WebCryptoPubkeyHex: "stored-public-key",
	}}}
	svc := NewEscrowService(repo, trusteeRepo, nil, bus)
	_, err := svc.SubmitTrusteeSignature(context.Background(), "task1", 7, "l1-signature", "l2-signature")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	bus.Replay()
	if !got {
		t.Fatal("expected ThresholdReached event to be published")
	}
}

func TestSubmitTrusteeSignatureDerivesStoredSignerKey(t *testing.T) {
	repo := &mockSignatureRepo{}
	trusteeRepo := &mockTrusteeRepo{keys: []trustee.TrusteeKey{{
		UserID:             7,
		WebCryptoPubkeyHex: "stored-public-key",
	}}}
	svc := NewEscrowService(repo, trusteeRepo, nil, nil)

	_, err := svc.SubmitTrusteeSignature(context.Background(), "task1", 7, "l1", "l2")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.saved == nil || repo.saved.TrusteePublicKeyHex != "stored-public-key" {
		t.Fatalf("expected signer key to come from trustee relationship, got %#v", repo.saved)
	}
}

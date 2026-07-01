package service

import (
    "context"
    "errors"
    "testing"

    "pamojabuild1/backend/internal/trustee"
)

type mockTrusteeUserRepo struct {
    created *trustee.User
    createErr error
}

func (m *mockTrusteeUserRepo) Create(ctx context.Context, u *trustee.User) error {
    if m.createErr != nil {
        return m.createErr
    }
    m.created = u
    return nil
}

func (m *mockTrusteeUserRepo) GetByID(ctx context.Context, id int64) (*trustee.User, error) {
    return nil, nil
}

func (m *mockTrusteeUserRepo) GetByEmail(ctx context.Context, email string) (*trustee.User, error) {
    return nil, nil
}

type mockTrusteeKeyRepo struct {
    saved    *trustee.TrusteeKey
    specific *trustee.TrusteeKey
    saveErr  error
}

func (m *mockTrusteeKeyRepo) SaveKeys(ctx context.Context, key *trustee.TrusteeKey) error {
    if m.saveErr != nil {
        return m.saveErr
    }
    m.saved = key
    return nil
}

func (m *mockTrusteeKeyRepo) GetKeysByTask(ctx context.Context, taskSlug string) ([]trustee.TrusteeKey, error) {
    return nil, nil
}

func (m *mockTrusteeKeyRepo) GetSpecificTrustee(ctx context.Context, taskSlug string, trusteeIndex int32) (*trustee.TrusteeKey, error) {
    return m.specific, nil
}

func TestRegisterUser(t *testing.T) {
    userRepo := &mockTrusteeUserRepo{}
    svc := NewTrusteeService(nil, userRepo)

    user, err := svc.RegisterUser(context.Background(), "alice@example.com", "secret", "Alice")
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if user == nil || user.Email != "alice@example.com" {
        t.Fatalf("expected user to be created, got %#v", user)
    }
    if userRepo.created == nil {
        t.Fatal("expected Create to be called on user repo")
    }
}

func TestAssignTrusteeSlot(t *testing.T) {
    keyRepo := &mockTrusteeKeyRepo{}
    svc := NewTrusteeService(keyRepo, nil)

    err := svc.AssignTrusteeSlot(context.Background(), "task1", &trustee.TrusteeKey{TrusteeIndex: 2})
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if keyRepo.saved == nil || keyRepo.saved.TrusteeIndex != 2 {
        t.Fatalf("expected trustee key saved, got %#v", keyRepo.saved)
    }
}

func TestAssignTrusteeSlotInvalidIndex(t *testing.T) {
    svc := NewTrusteeService(nil, nil)
    err := svc.AssignTrusteeSlot(context.Background(), "task1", &trustee.TrusteeKey{TrusteeIndex: 5})
    if !errors.Is(err, ErrInvalidTrusteeIndex) {
        t.Fatalf("expected ErrInvalidTrusteeIndex, got %v", err)
    }
}

func TestAssignTrusteeSlotTaken(t *testing.T) {
    keyRepo := &mockTrusteeKeyRepo{specific: &trustee.TrusteeKey{TrusteeIndex: 1, UserID: 42}}
    svc := NewTrusteeService(keyRepo, nil)

    err := svc.AssignTrusteeSlot(context.Background(), "task1", &trustee.TrusteeKey{TrusteeIndex: 1})
    if !errors.Is(err, ErrSlotAlreadyTaken) {
        t.Fatalf("expected ErrSlotAlreadyTaken, got %v", err)
    }
}

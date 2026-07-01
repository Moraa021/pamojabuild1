package service

import (
    "context"
    "errors"
    "testing"

    "golang.org/x/crypto/bcrypt"

    "pamojabuild1/backend/internal/auth"
)

type mockAuthRepo struct {
    users        map[string]*auth.User
    usersByID    map[int64]*auth.User
    createErr    error
    getByPhoneErr error
    getByIDErr   error
}

func newMockAuthRepo() *mockAuthRepo {
    return &mockAuthRepo{
        users:     make(map[string]*auth.User),
        usersByID: make(map[int64]*auth.User),
    }
}

func (m *mockAuthRepo) Create(ctx context.Context, user *auth.User) error {
    if m.createErr != nil {
        return m.createErr
    }
    if _, exists := m.users[user.PhoneNumber]; exists {
        return errors.New("duplicate")
    }
    user.ID = int64(len(m.users) + 1)
    m.users[user.PhoneNumber] = user
    m.usersByID[user.ID] = user
    return nil
}

func (m *mockAuthRepo) GetByID(ctx context.Context, id int64) (*auth.User, error) {
    if m.getByIDErr != nil {
        return nil, m.getByIDErr
    }
    user, ok := m.usersByID[id]
    if !ok {
        return nil, errors.New("not found")
    }
    return user, nil
}

func (m *mockAuthRepo) GetByPhone(ctx context.Context, phone string) (*auth.User, error) {
    if m.getByPhoneErr != nil {
        return nil, m.getByPhoneErr
    }
    user, ok := m.users[phone]
    if !ok {
        return nil, errors.New("not found")
    }
    return user, nil
}

func (m *mockAuthRepo) UpdateRole(ctx context.Context, userID int64, role string) error {
    user, ok := m.usersByID[userID]
    if !ok {
        return errors.New("not found")
    }
    user.Role = role
    return nil
}

func TestRegisterSuccess(t *testing.T) {
    repo := newMockAuthRepo()
    svc := NewAuthService(repo, "test-secret")

    user, token, err := svc.Register(context.Background(), "+15551234567", "password123", "Alice")
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if user == nil {
        t.Fatal("expected user returned")
    }
    if user.PhoneNumber != "+15551234567" {
        t.Fatalf("expected phone number +15551234567, got %s", user.PhoneNumber)
    }
    if user.Role != "volunteer" {
        t.Fatalf("expected volunteer role, got %s", user.Role)
    }
    if token == "" {
        t.Fatal("expected token to be generated")
    }
}

func TestRegisterExistingUser(t *testing.T) {
    repo := newMockAuthRepo()
    repo.users["+15559876543"] = &auth.User{ID: 1, PhoneNumber: "+15559876543"}
    svc := NewAuthService(repo, "test-secret")

    _, _, err := svc.Register(context.Background(), "+15559876543", "password123", "Bob")
    if !errors.Is(err, ErrUserExists) {
        t.Fatalf("expected ErrUserExists, got %v", err)
    }
}

func TestSignInSuccess(t *testing.T) {
    repo := newMockAuthRepo()
    hashed, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
    if err != nil {
        t.Fatalf("failed to hash password: %v", err)
    }
    repo.users["+15557654321"] = &auth.User{ID: 2, PhoneNumber: "+15557654321", PasswordHash: string(hashed), Role: "volunteer"}
    repo.usersByID[2] = repo.users["+15557654321"]
    svc := NewAuthService(repo, "test-secret")

    user, token, err := svc.SignIn(context.Background(), "+15557654321", "password123")
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if user.PhoneNumber != "+15557654321" {
        t.Fatalf("expected phone number +15557654321, got %s", user.PhoneNumber)
    }
    if token == "" {
        t.Fatal("expected token to be generated")
    }
}

func TestSignInInvalidPassword(t *testing.T) {
    repo := newMockAuthRepo()
    hashed, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
    if err != nil {
        t.Fatalf("failed to hash password: %v", err)
    }
    repo.users["+15553456789"] = &auth.User{ID: 3, PhoneNumber: "+15553456789", PasswordHash: string(hashed)}
    repo.usersByID[3] = repo.users["+15553456789"]
    svc := NewAuthService(repo, "test-secret")

    _, _, err = svc.SignIn(context.Background(), "+15553456789", "wrongpass")
    if !errors.Is(err, ErrInvalidCredentials) {
        t.Fatalf("expected ErrInvalidCredentials, got %v", err)
    }
}

func TestValidateTokenSuccess(t *testing.T) {
    repo := newMockAuthRepo()
    user := &auth.User{ID: 4, PhoneNumber: "+15550001122", Role: "volunteer"}
    repo.usersByID[4] = user
    svc := NewAuthService(repo, "test-secret")

    token, err := svc.generateToken(user)
    if err != nil {
        t.Fatalf("expected token generation to succeed, got %v", err)
    }

    validatedUser, err := svc.ValidateToken(context.Background(), token)
    if err != nil {
        t.Fatalf("expected no error validating token, got %v", err)
    }
    if validatedUser.ID != 4 {
        t.Fatalf("expected user ID 4, got %d", validatedUser.ID)
    }
}

func TestValidateTokenInvalid(t *testing.T) {
    repo := newMockAuthRepo()
    svc := NewAuthService(repo, "test-secret")

    _, err := svc.ValidateToken(context.Background(), "invalid-token")
    if !errors.Is(err, ErrInvalidToken) {
        t.Fatalf("expected ErrInvalidToken, got %v", err)
    }
}

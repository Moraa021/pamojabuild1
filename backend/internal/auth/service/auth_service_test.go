package service

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"pamojabuild1/backend/internal/auth"
)

const testJWTSecret = "test-secret-that-is-at-least-32-bytes"

type mockAuthRepo struct {
	users     map[string]*auth.User
	usersByID map[int64]*auth.User
	createErr error
}

func newMockAuthRepo() *mockAuthRepo {
	return &mockAuthRepo{
		users:     make(map[string]*auth.User),
		usersByID: make(map[int64]*auth.User),
	}
}

func (m *mockAuthRepo) Create(_ context.Context, user *auth.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	if _, exists := m.users[user.PhoneNumber]; exists {
		return auth.ErrPhoneNumberTaken
	}
	user.ID = int64(len(m.users) + 1)
	m.users[user.PhoneNumber] = user
	m.usersByID[user.ID] = user
	return nil
}

func (m *mockAuthRepo) GetByID(_ context.Context, id int64) (*auth.User, error) {
	user, ok := m.usersByID[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return user, nil
}

func (m *mockAuthRepo) GetByPhone(_ context.Context, phone string) (*auth.User, error) {
	user, ok := m.users[phone]
	if !ok {
		return nil, errors.New("not found")
	}
	return user, nil
}

func (m *mockAuthRepo) IncrementTokenVersion(_ context.Context, userID int64) error {
	user, ok := m.usersByID[userID]
	if !ok {
		return errors.New("not found")
	}
	user.TokenVersion++
	return nil
}

func newTestService(t *testing.T, repo auth.Repository) *AuthService {
	t.Helper()
	service, err := NewAuthService(repo, testJWTSecret)
	if err != nil {
		t.Fatalf("create auth service: %v", err)
	}
	return service
}

func TestNewAuthServiceRejectsUnsafeSecret(t *testing.T) {
	if _, err := NewAuthService(newMockAuthRepo(), "short"); !errors.Is(err, ErrUnsafeJWTSecret) {
		t.Fatalf("expected ErrUnsafeJWTSecret, got %v", err)
	}
}

func TestRegisterCreatesGeneralAccountAndNormalizesPhone(t *testing.T) {
	repo := newMockAuthRepo()
	service := newTestService(t, repo)

	user, token, err := service.Register(context.Background(), "+1 (555) 123-4567", "password123", " Alice ")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if user.PhoneNumber != "+15551234567" {
		t.Fatalf("expected normalized phone, got %q", user.PhoneNumber)
	}
	if user.DisplayName != "Alice" {
		t.Fatalf("expected trimmed display name, got %q", user.DisplayName)
	}
	if user.IsAdmin {
		t.Fatal("new account must not be an administrator")
	}
	if token == "" {
		t.Fatal("expected token")
	}
}

func TestRegisterRejectsDuplicatePhoneAfterNormalization(t *testing.T) {
	repo := newMockAuthRepo()
	service := newTestService(t, repo)
	if _, _, err := service.Register(context.Background(), "+15551234567", "password123", "Alice"); err != nil {
		t.Fatalf("first register: %v", err)
	}
	if _, _, err := service.Register(context.Background(), "+1 555 123 4567", "password123", "Other"); !errors.Is(err, ErrUserExists) {
		t.Fatalf("expected ErrUserExists, got %v", err)
	}
}

func TestSignInAndValidateToken(t *testing.T) {
	repo := newMockAuthRepo()
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := &auth.User{ID: 4, PhoneNumber: "+15550001122", PasswordHash: string(hash)}
	repo.users[user.PhoneNumber] = user
	repo.usersByID[user.ID] = user
	service := newTestService(t, repo)

	_, token, err := service.SignIn(context.Background(), "+1 (555) 000-1122", "password123")
	if err != nil {
		t.Fatalf("sign in: %v", err)
	}
	validated, err := service.ValidateToken(context.Background(), token)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if validated.ID != user.ID {
		t.Fatalf("expected user %d, got %d", user.ID, validated.ID)
	}
}

func TestSignOutRevokesPreviouslyIssuedToken(t *testing.T) {
	repo := newMockAuthRepo()
	user := &auth.User{ID: 7, PhoneNumber: "+15550000007"}
	repo.usersByID[user.ID] = user
	service := newTestService(t, repo)
	token, err := service.generateToken(user)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	if err := service.SignOut(context.Background(), user.ID); err != nil {
		t.Fatalf("sign out: %v", err)
	}
	if _, err := service.ValidateToken(context.Background(), token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected revoked token to be invalid, got %v", err)
	}
}

func TestValidateTokenRejectsMalformedTokenWithoutPanicking(t *testing.T) {
	service := newTestService(t, newMockAuthRepo())
	if _, err := service.ValidateToken(context.Background(), "invalid-token"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"pamojabuild1/backend/internal/auth"
)

type mockAuthRepo struct {
	users        map[string]*auth.User
	usersByID    map[int64]*auth.User
	sessionUsers map[string]*auth.User
	createErr    error
}

func newMockAuthRepo() *mockAuthRepo {
	return &mockAuthRepo{
		users:        make(map[string]*auth.User),
		usersByID:    make(map[int64]*auth.User),
		sessionUsers: make(map[string]*auth.User),
	}
}

func (m *mockAuthRepo) Create(_ context.Context, user *auth.User, session *auth.Session) error {
	if m.createErr != nil {
		return m.createErr
	}
	if _, exists := m.users[user.PhoneNumber]; exists {
		return auth.ErrPhoneNumberTaken
	}
	user.ID = int64(len(m.users) + 1)
	m.users[user.PhoneNumber] = user
	m.usersByID[user.ID] = user
	m.sessionUsers[string(session.TokenHash)] = user
	return nil
}

func (m *mockAuthRepo) GetByID(_ context.Context, id int64) (*auth.User, error) {
	user, ok := m.usersByID[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return user, nil
}

func (m *mockAuthRepo) GetByPhone(_ context.Context, phone string) (*auth.User, error) {
	user, ok := m.users[phone]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return user, nil
}

func (m *mockAuthRepo) CreateSession(_ context.Context, userID int64, session *auth.Session) error {
	user, ok := m.usersByID[userID]
	if !ok {
		return sql.ErrNoRows
	}
	m.sessionUsers[string(session.TokenHash)] = user
	return nil
}

func (m *mockAuthRepo) GetBySessionHash(_ context.Context, tokenHash []byte) (*auth.User, error) {
	user, ok := m.sessionUsers[string(tokenHash)]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return user, nil
}

func (m *mockAuthRepo) RevokeSession(_ context.Context, tokenHash []byte) error {
	delete(m.sessionUsers, string(tokenHash))
	return nil
}

func TestRegisterCreatesGeneralAccountAndSession(t *testing.T) {
	repo := newMockAuthRepo()
	service := NewAuthService(repo)

	session, err := service.Register(context.Background(), "+1 (555) 123-4567", "password123", " Alice ")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if session.User.PhoneNumber != "+15551234567" {
		t.Fatalf("expected normalized phone, got %q", session.User.PhoneNumber)
	}
	if session.User.DisplayName != "Alice" {
		t.Fatalf("expected trimmed display name, got %q", session.User.DisplayName)
	}
	if session.User.IsAdmin {
		t.Fatal("new account must not be an administrator")
	}
	if session.Token == "" || len(session.TokenHash) != sha256.Size {
		t.Fatal("expected random token and SHA-256 token hash")
	}
	if session.ExpiresAt.IsZero() {
		t.Fatal("expected session expiry")
	}
}

func TestRegisterRejectsDuplicatePhoneAfterNormalization(t *testing.T) {
	repo := newMockAuthRepo()
	service := NewAuthService(repo)
	if _, err := service.Register(context.Background(), "+15551234567", "password123", "Alice"); err != nil {
		t.Fatalf("first register: %v", err)
	}
	if _, err := service.Register(context.Background(), "+1 555 123 4567", "password123", "Other"); !errors.Is(err, ErrUserExists) {
		t.Fatalf("expected ErrUserExists, got %v", err)
	}
}

func TestSignInAndAuthenticateSession(t *testing.T) {
	repo := newMockAuthRepo()
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := &auth.User{ID: 4, PhoneNumber: "+15550001122", PasswordHash: string(hash)}
	repo.users[user.PhoneNumber] = user
	repo.usersByID[user.ID] = user
	service := NewAuthService(repo)

	session, err := service.SignIn(context.Background(), "+1 (555) 000-1122", "password123")
	if err != nil {
		t.Fatalf("sign in: %v", err)
	}
	validated, err := service.Authenticate(context.Background(), session.Token)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if validated.ID != user.ID {
		t.Fatalf("expected user %d, got %d", user.ID, validated.ID)
	}
}

func TestSignOutRevokesOnlyPresentedSession(t *testing.T) {
	repo := newMockAuthRepo()
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := &auth.User{ID: 7, PhoneNumber: "+15550000007", PasswordHash: string(hash)}
	repo.users[user.PhoneNumber] = user
	repo.usersByID[user.ID] = user
	service := NewAuthService(repo)

	first, err := service.SignIn(context.Background(), user.PhoneNumber, "password123")
	if err != nil {
		t.Fatalf("first sign in: %v", err)
	}
	second, err := service.SignIn(context.Background(), user.PhoneNumber, "password123")
	if err != nil {
		t.Fatalf("second sign in: %v", err)
	}
	if err := service.SignOut(context.Background(), first.Token); err != nil {
		t.Fatalf("sign out: %v", err)
	}
	if _, err := service.Authenticate(context.Background(), first.Token); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("expected first session revoked, got %v", err)
	}
	if _, err := service.Authenticate(context.Background(), second.Token); err != nil {
		t.Fatalf("expected second session to remain active, got %v", err)
	}
}

func TestAuthenticateRejectsMalformedTokenWithoutPanicking(t *testing.T) {
	service := NewAuthService(newMockAuthRepo())
	if _, err := service.Authenticate(context.Background(), "invalid-token"); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("expected ErrInvalidSession, got %v", err)
	}
}

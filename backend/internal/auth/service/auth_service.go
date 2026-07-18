package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"golang.org/x/crypto/bcrypt"

	"pamojabuild1/backend/internal/auth"
)

const sessionTTL = 24 * time.Hour

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidSession     = errors.New("invalid or expired session")
	ErrInvalidPhone       = errors.New("phone number must use international format, for example +254700000000")
	ErrWeakPassword       = errors.New("password must contain at least 8 characters")
	ErrInvalidDisplayName = errors.New("display name is required")
)

type AuthService struct {
	repo auth.Repository
}

func NewAuthService(repo auth.Repository) *AuthService {
	return &AuthService{repo: repo}
}

func (s *AuthService) Register(ctx context.Context, phone, password, displayName string) (*auth.Session, error) {
	normalizedPhone, err := normalizePhoneNumber(phone)
	if err != nil {
		return nil, err
	}
	if len(password) < 8 {
		return nil, ErrWeakPassword
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return nil, ErrInvalidDisplayName
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	session, err := newSession()
	if err != nil {
		return nil, err
	}
	user := &auth.User{
		PhoneNumber:  normalizedPhone,
		PasswordHash: string(hashedPassword),
		DisplayName:  displayName,
	}
	session.User = user

	// Account, optional profile, and first session are committed together. A
	// partial registration must never leave an account the caller cannot use.
	if err := s.repo.Create(ctx, user, session); err != nil {
		if errors.Is(err, auth.ErrPhoneNumberTaken) {
			return nil, ErrUserExists
		}
		return nil, fmt.Errorf("create account: %w", err)
	}
	return session, nil
}

func (s *AuthService) SignIn(ctx context.Context, phone, password string) (*auth.Session, error) {
	normalizedPhone, err := normalizePhoneNumber(phone)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	user, err := s.repo.GetByPhone(ctx, normalizedPhone)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("load account for sign in: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	session, err := newSession()
	if err != nil {
		return nil, err
	}
	session.User = user
	if err := s.repo.CreateSession(ctx, user.ID, session); err != nil {
		return nil, fmt.Errorf("persist sign-in session: %w", err)
	}
	return session, nil
}

func (s *AuthService) SignOut(ctx context.Context, token string) error {
	hash, err := hashSessionToken(token)
	if err != nil {
		// An invalid or already-cleared cookie is already signed out.
		return nil
	}
	if err := s.repo.RevokeSession(ctx, hash); err != nil {
		return fmt.Errorf("sign out: %w", err)
	}
	return nil
}

func (s *AuthService) Authenticate(ctx context.Context, token string) (*auth.User, error) {
	hash, err := hashSessionToken(token)
	if err != nil {
		return nil, ErrInvalidSession
	}
	user, err := s.repo.GetBySessionHash(ctx, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidSession
		}
		return nil, fmt.Errorf("load account session: %w", err)
	}
	return user, nil
}

func newSession() (*auth.Session, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("generate session token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	hash := sha256.Sum256([]byte(token))
	return &auth.Session{
		Token:     token,
		TokenHash: hash[:],
		ExpiresAt: time.Now().Add(sessionTTL),
	}, nil
}

func hashSessionToken(token string) ([]byte, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(decoded) != 32 {
		return nil, ErrInvalidSession
	}
	hash := sha256.Sum256([]byte(token))
	return hash[:], nil
}

// normalizePhoneNumber stores one canonical form so formatting differences
// cannot produce duplicate accounts. This deliberately accepts common visual
// separators but requires an international '+' prefix and 8-15 digits.
func normalizePhoneNumber(value string) (string, error) {
	value = strings.TrimSpace(value)
	var digits strings.Builder
	for index, r := range value {
		switch {
		case r == '+' && index == 0:
		case r >= '0' && r <= '9':
			digits.WriteRune(r)
		case unicode.IsSpace(r) || r == '-' || r == '(' || r == ')':
		default:
			return "", ErrInvalidPhone
		}
	}
	if !strings.HasPrefix(value, "+") || digits.Len() < 8 || digits.Len() > 15 {
		return "", ErrInvalidPhone
	}
	return "+" + digits.String(), nil
}

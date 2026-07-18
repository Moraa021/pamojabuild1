package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"pamojabuild1/backend/internal/auth"
)

const (
	tokenIssuer = "pamojabuild"
	tokenTTL    = 24 * time.Hour
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidToken       = errors.New("invalid token")
	ErrInvalidPhone       = errors.New("phone number must use international format, for example +254700000000")
	ErrWeakPassword       = errors.New("password must contain at least 8 characters")
	ErrInvalidDisplayName = errors.New("display name is required")
	ErrUnsafeJWTSecret    = errors.New("JWT secret must contain at least 32 characters")
)

type tokenClaims struct {
	UserID       int64 `json:"uid"`
	TokenVersion int64 `json:"ver"`
	jwt.RegisteredClaims
}

type AuthService struct {
	repo      auth.Repository
	jwtSecret []byte
}

func NewAuthService(repo auth.Repository, jwtSecret string) (*AuthService, error) {
	if len(jwtSecret) < 32 {
		return nil, ErrUnsafeJWTSecret
	}
	return &AuthService{repo: repo, jwtSecret: []byte(jwtSecret)}, nil
}

func (s *AuthService) Register(ctx context.Context, phone, password, displayName string) (*auth.User, string, error) {
	normalizedPhone, err := normalizePhoneNumber(phone)
	if err != nil {
		return nil, "", err
	}
	if len(password) < 8 {
		return nil, "", ErrWeakPassword
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return nil, "", ErrInvalidDisplayName
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", fmt.Errorf("hash password: %w", err)
	}

	user := &auth.User{
		PhoneNumber:  normalizedPhone,
		PasswordHash: string(hashedPassword),
		DisplayName:  displayName,
	}
	if err := s.repo.Create(ctx, user); err != nil {
		if errors.Is(err, auth.ErrPhoneNumberTaken) {
			return nil, "", ErrUserExists
		}
		return nil, "", fmt.Errorf("create account: %w", err)
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
}

func (s *AuthService) SignIn(ctx context.Context, phone, password string) (*auth.User, string, error) {
	normalizedPhone, err := normalizePhoneNumber(phone)
	if err != nil {
		return nil, "", ErrInvalidCredentials
	}

	user, err := s.repo.GetByPhone(ctx, normalizedPhone)
	if err != nil {
		return nil, "", ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", ErrInvalidCredentials
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
}

// SignOut increments a version stored with the account. Every JWT carries the
// version it was issued under, so this invalidates all outstanding tokens even
// though JWTs are otherwise self-contained and cannot be deleted server-side.
func (s *AuthService) SignOut(ctx context.Context, userID int64) error {
	if err := s.repo.IncrementTokenVersion(ctx, userID); err != nil {
		return fmt.Errorf("sign out: %w", err)
	}
	return nil
}

func (s *AuthService) generateToken(user *auth.User) (string, error) {
	now := time.Now()
	claims := tokenClaims{
		UserID:       user.ID,
		TokenVersion: user.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tokenIssuer,
			Subject:   strconv.FormatInt(user.ID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("sign authentication token: %w", err)
	}
	return signed, nil
}

func (s *AuthService) ValidateToken(ctx context.Context, tokenString string) (*auth.User, error) {
	claims := &tokenClaims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			return s.jwtSecret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(tokenIssuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !token.Valid || claims.UserID <= 0 {
		return nil, ErrInvalidToken
	}

	user, err := s.repo.GetByID(ctx, claims.UserID)
	if err != nil || user.TokenVersion != claims.TokenVersion {
		return nil, ErrInvalidToken
	}
	return user, nil
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

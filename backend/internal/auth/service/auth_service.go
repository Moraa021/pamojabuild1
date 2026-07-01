package service

import (
	"context"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidToken       = errors.New("invalid token")
)

type AuthService struct {
	repo      auth.Repository
	jwtSecret string
}

func NewAuthService(repo auth.Repository, jwtSecret string) *AuthService {
	return &AuthService{repo: repo, jwtSecret: jwtSecret}
}

func (s *AuthService) Register(ctx context.Context, email, password, displayName string) (*auth.User, string, error) {
	existing, _ := s.repo.GetByEmail(ctx, email)
	if existing != nil {
		return nil, "", ErrUserExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	user := &auth.User{
		Email:        email,
		PasswordHash: string(hashedPassword),
		DisplayName:  displayName,
		Role:         "volunteer", // default role
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, "", err
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (s *AuthService) SignIn(ctx context.Context, email, password string) (*auth.User, string, error) {
	user, err := s.repo.GetByEmail(ctx, email)
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

func (s *AuthService) generateToken(user *auth.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    user.Role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *AuthService) ValidateToken(ctx context.Context, tokenString string) (*auth.User, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	userID := int64(claims["user_id"].(float64))
	return s.repo.GetByID(ctx, userID)
}
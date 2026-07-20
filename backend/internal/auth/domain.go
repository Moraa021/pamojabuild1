package auth

import (
	"context"
	"errors"
	"time"
)

var ErrPhoneNumberTaken = errors.New("phone number already registered")

type User struct {
	ID           int64
	PhoneNumber  string
	PasswordHash string
	DisplayName  string
	IsAdmin      bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Session struct {
	User      *User
	Token     string
	TokenHash []byte
	ExpiresAt time.Time
}

type Repository interface {
	Create(ctx context.Context, user *User, session *Session) error
	GetByID(ctx context.Context, id int64) (*User, error)
	GetByPhone(ctx context.Context, phone string) (*User, error)
	CreateSession(ctx context.Context, userID int64, session *Session) error
	GetBySessionHash(ctx context.Context, tokenHash []byte) (*User, error)
	RevokeSession(ctx context.Context, tokenHash []byte) error
}

type Service interface {
	Register(ctx context.Context, phone, password, displayName string) (*Session, error)
	SignIn(ctx context.Context, phone, password string) (*Session, error)
	SignOut(ctx context.Context, token string) error
	Authenticate(ctx context.Context, token string) (*User, error)
}

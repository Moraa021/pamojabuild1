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
	TokenVersion int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Repository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id int64) (*User, error)
	GetByPhone(ctx context.Context, phone string) (*User, error)
	IncrementTokenVersion(ctx context.Context, userID int64) error
}

type Service interface {
	Register(ctx context.Context, phone, password, displayName string) (*User, string, error)
	SignIn(ctx context.Context, phone, password string) (*User, string, error)
	SignOut(ctx context.Context, userID int64) error
	ValidateToken(ctx context.Context, token string) (*User, error)
}

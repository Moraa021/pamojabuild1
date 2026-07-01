package auth

import (
	"context"
	"time"
)

type User struct {
	ID           int64
	PhoneNumber  string
	PasswordHash string
	DisplayName  string
	Role         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Repository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id int64) (*User, error)
	GetByPhone(ctx context.Context, phone string) (*User, error)
	UpdateRole(ctx context.Context, userID int64, role string) error
}

type Service interface {
	Register(ctx context.Context, phone, password, displayName string) (*User, string, error)
	SignIn(ctx context.Context, phone, password string) (*User, string, error)
	SignOut(ctx context.Context, token string) error
	ValidateToken(ctx context.Context, token string) (*User, error)
}
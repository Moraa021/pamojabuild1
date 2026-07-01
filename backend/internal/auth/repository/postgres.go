package repository

import (
	"context"
	"database/sql"
	"time"

	"pamojabuild1/backend/internal/auth"
)

type AuthRepository struct {
	db *sql.DB
}

func NewAuthRepository(db *sql.DB) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) Create(ctx context.Context, user *auth.User) error {
	query := `
		INSERT INTO users (email, password_hash, display_name, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	return r.db.QueryRowContext(ctx, query,
		user.PhoneNumber, user.PasswordHash, user.DisplayName, user.Role, now, now,
	).Scan(&user.ID)
}

func (r *AuthRepository) GetByID(ctx context.Context, id int64) (*auth.User, error) {
	user := &auth.User{}
	query := `SELECT id, email, password_hash, display_name, role, created_at, updated_at FROM users WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.PhoneNumber, &user.PasswordHash, &user.DisplayName, &user.Role, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *AuthRepository) GetByPhone(ctx context.Context, phone string) (*auth.User, error) {
	user := &auth.User{}
	query := `SELECT id, email, password_hash, display_name, role, created_at, updated_at FROM users WHERE email = $1`
	err := r.db.QueryRowContext(ctx, query, phone).Scan(
		&user.ID, &user.PhoneNumber, &user.PasswordHash, &user.DisplayName, &user.Role, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}
func (r *AuthRepository) UpdateRole(ctx context.Context, userID int64, role string) error {
	query := `UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, role, userID)
	return err
}

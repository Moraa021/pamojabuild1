package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"

	"pamojabuild1/backend/internal/auth"
)

type AuthRepository struct {
	db *sql.DB
}

func NewAuthRepository(db *sql.DB) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) Create(ctx context.Context, user *auth.User) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin account creation: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO users (phone_number, password_hash, display_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, is_admin, token_version`

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	if err := tx.QueryRowContext(ctx, query,
		user.PhoneNumber, user.PasswordHash, user.DisplayName, now, now,
	).Scan(&user.ID, &user.IsAdmin, &user.TokenVersion); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return auth.ErrPhoneNumberTaken
		}
		return fmt.Errorf("insert account: %w", err)
	}

	// Every account can participate as a volunteer on a future task, so its
	// optional profile is created atomically with the account rather than being
	// treated as a permanent global "volunteer" role.
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO volunteer_profiles (user_id, created_at, updated_at) VALUES ($1, $2, $2)`,
		user.ID, now,
	); err != nil {
		return fmt.Errorf("create account profile: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit account creation: %w", err)
	}
	return nil
}

func (r *AuthRepository) GetByID(ctx context.Context, id int64) (*auth.User, error) {
	user := &auth.User{}
	query := `SELECT id, phone_number, password_hash, display_name, is_admin, token_version, created_at, updated_at FROM users WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.PhoneNumber, &user.PasswordHash, &user.DisplayName, &user.IsAdmin, &user.TokenVersion, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *AuthRepository) GetByPhone(ctx context.Context, phone string) (*auth.User, error) {
	user := &auth.User{}
	query := `SELECT id, phone_number, password_hash, display_name, is_admin, token_version, created_at, updated_at FROM users WHERE phone_number = $1`
	err := r.db.QueryRowContext(ctx, query, phone).Scan(
		&user.ID, &user.PhoneNumber, &user.PasswordHash, &user.DisplayName, &user.IsAdmin, &user.TokenVersion, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *AuthRepository) IncrementTokenVersion(ctx context.Context, userID int64) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE users SET token_version = token_version + 1, updated_at = NOW() WHERE id = $1`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("revoke account sessions: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read revoked session count: %w", err)
	}
	if affected != 1 {
		return sql.ErrNoRows
	}
	return nil
}

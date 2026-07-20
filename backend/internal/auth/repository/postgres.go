package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"pamojabuild1/backend/internal/auth"
)

type AuthRepository struct {
	db *sql.DB
}

func NewAuthRepository(db *sql.DB) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) Create(ctx context.Context, user *auth.User, session *auth.Session) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin account creation: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO users (phone_number, password_hash, display_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, is_admin`

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	if err := tx.QueryRowContext(ctx, query,
		user.PhoneNumber, user.PasswordHash, user.DisplayName, now, now,
	).Scan(&user.ID, &user.IsAdmin); err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == "23505" {
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

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO user_sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`,
		user.ID, session.TokenHash, session.ExpiresAt,
	); err != nil {
		return fmt.Errorf("create initial account session: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit account creation: %w", err)
	}
	return nil
}

func (r *AuthRepository) GetByID(ctx context.Context, id int64) (*auth.User, error) {
	user := &auth.User{}
	query := `SELECT id, phone_number, password_hash, display_name, is_admin, created_at, updated_at FROM users WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.PhoneNumber, &user.PasswordHash, &user.DisplayName, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *AuthRepository) GetByPhone(ctx context.Context, phone string) (*auth.User, error) {
	user := &auth.User{}
	query := `SELECT id, phone_number, password_hash, display_name, is_admin, created_at, updated_at FROM users WHERE phone_number = $1`
	err := r.db.QueryRowContext(ctx, query, phone).Scan(
		&user.ID, &user.PhoneNumber, &user.PasswordHash, &user.DisplayName, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *AuthRepository) CreateSession(ctx context.Context, userID int64, session *auth.Session) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO user_sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`,
		userID, session.TokenHash, session.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("create user session: %w", err)
	}
	return nil
}

func (r *AuthRepository) GetBySessionHash(ctx context.Context, tokenHash []byte) (*auth.User, error) {
	user := &auth.User{}
	err := r.db.QueryRowContext(ctx, `
		SELECT u.id, u.phone_number, u.password_hash, u.display_name,
		       u.is_admin, u.created_at, u.updated_at
		FROM user_sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1
		  AND s.revoked_at IS NULL
		  AND s.expires_at > NOW()`,
		tokenHash,
	).Scan(
		&user.ID, &user.PhoneNumber, &user.PasswordHash, &user.DisplayName,
		&user.IsAdmin, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *AuthRepository) RevokeSession(ctx context.Context, tokenHash []byte) error {
	// Revocation is deliberately idempotent. Repeating sign-out must not turn a
	// successful security action into an error because the session is gone.
	if _, err := r.db.ExecContext(ctx, `
		UPDATE user_sessions
		SET revoked_at = COALESCE(revoked_at, NOW())
		WHERE token_hash = $1`,
		tokenHash,
	); err != nil {
		return fmt.Errorf("revoke user session: %w", err)
	}
	return nil
}

package repository

import (
	"context"
	"database/sql"

	"pamojabuild1/backend/internal/trustee"
)

type TrusteeRepository struct {
	db *sql.DB
}

func NewTrusteeRepository(db *sql.DB) *TrusteeRepository {
	return &TrusteeRepository{db: db}
}

func (r *TrusteeRepository) SaveKeys(ctx context.Context, key *trustee.TrusteeKey) error {
	query := `
		INSERT INTO trustee_keys (task_slug, trustee_index, user_id, xpub, web_crypto_pubkey_hex)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (task_slug, trustee_index) DO UPDATE
		SET user_id = $3, xpub = $4, web_crypto_pubkey_hex = $5`

	_, err := r.db.ExecContext(ctx, query,
		key.TaskSlug, key.TrusteeIndex, key.UserID, key.Xpub, key.WebCryptoPubkeyHex,
	)
	return err
}

func (r *TrusteeRepository) GetKeysByTask(ctx context.Context, taskSlug string) ([]trustee.TrusteeKey, error) {
	query := `
		SELECT task_slug, trustee_index, user_id, xpub, web_crypto_pubkey_hex
		FROM trustee_keys WHERE task_slug = $1
		ORDER BY trustee_index`

	rows, err := r.db.QueryContext(ctx, query, taskSlug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []trustee.TrusteeKey
	for rows.Next() {
		var key trustee.TrusteeKey
		if err := rows.Scan(&key.TaskSlug, &key.TrusteeIndex, &key.UserID,
			&key.Xpub, &key.WebCryptoPubkeyHex); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

func (r *TrusteeRepository) GetSpecificTrustee(ctx context.Context, taskSlug string, trusteeIndex int32) (*trustee.TrusteeKey, error) {
	key := &trustee.TrusteeKey{}
	query := `
		SELECT task_slug, trustee_index, user_id, xpub, web_crypto_pubkey_hex
		FROM trustee_keys WHERE task_slug = $1 AND trustee_index = $2`

	err := r.db.QueryRowContext(ctx, query, taskSlug, trusteeIndex).Scan(
		&key.TaskSlug, &key.TrusteeIndex, &key.UserID, &key.Xpub, &key.WebCryptoPubkeyHex,
	)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (r *TrusteeRepository) CreateUser(ctx context.Context, u *trustee.User) error {
	query := `
		INSERT INTO users (email, password_hash, display_name)
		VALUES ($1, $2, $3)
		RETURNING id`

	return r.db.QueryRowContext(ctx, query,
		u.Email, u.PasswordHash, u.DisplayName,
	).Scan(&u.ID)
}

func (r *TrusteeRepository) GetByID(ctx context.Context, id int64) (*trustee.User, error) {
	u := &trustee.User{}
	query := `SELECT id, email, password_hash, display_name FROM users WHERE id = $1`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName,
	)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *TrusteeRepository) GetByEmail(ctx context.Context, email string) (*trustee.User, error) {
	u := &trustee.User{}
	query := `SELECT id, email, password_hash, display_name FROM users WHERE email = $1`

	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName,
	)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *TrusteeRepository) Create(ctx context.Context, u *trustee.User) error {
	return r.CreateUser(ctx, u)
}
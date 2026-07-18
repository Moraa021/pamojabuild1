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
		VALUES ($1, $2, $3, $4, $5)`

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

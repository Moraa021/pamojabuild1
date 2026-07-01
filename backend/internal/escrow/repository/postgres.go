package repository

import (
	"context"
	"database/sql"

	"pamojabuild1/backend/internal/escrow"
)

type EscrowRepository struct {
	db *sql.DB
}

func NewEscrowRepository(db *sql.DB) *EscrowRepository {
	return &EscrowRepository{db: db}
}

func (r *EscrowRepository) SaveSignature(ctx context.Context, sig *escrow.SignatureCollection) error {
	query := `
		INSERT INTO payout_signatures (task_slug, trustee_public_key_hex, l1_signature_fragment, l2_web_crypto_signature)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (task_slug, trustee_public_key_hex) DO UPDATE
		SET l1_signature_fragment = $3, l2_web_crypto_signature = $4`

	_, err := r.db.ExecContext(ctx, query,
		sig.TaskSlug, sig.TrusteePublicKeyHex, sig.L1SignatureFragment, sig.L2WebCryptoSignature,
	)
	return err
}

func (r *EscrowRepository) GetSignatures(ctx context.Context, taskSlug string) ([]escrow.SignatureCollection, error) {
	query := `
		SELECT task_slug, trustee_public_key_hex, l1_signature_fragment, l2_web_crypto_signature
		FROM payout_signatures WHERE task_slug = $1`

	rows, err := r.db.QueryContext(ctx, query, taskSlug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var signatures []escrow.SignatureCollection
	for rows.Next() {
		var sig escrow.SignatureCollection
		if err := rows.Scan(&sig.TaskSlug, &sig.TrusteePublicKeyHex,
			&sig.L1SignatureFragment, &sig.L2WebCryptoSignature); err != nil {
			return nil, err
		}
		signatures = append(signatures, sig)
	}
	return signatures, nil
}

func (r *EscrowRepository) GetSignatureCount(ctx context.Context, taskSlug string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM payout_signatures WHERE task_slug = $1`
	err := r.db.QueryRowContext(ctx, query, taskSlug).Scan(&count)
	return count, err
}
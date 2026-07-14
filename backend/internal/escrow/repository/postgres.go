package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"pamojabuild1/internal/escrow"
)

type PostgresRepository struct {
	DB *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{DB: db}
}

func (r *PostgresRepository) SaveSignatureFragment(ctx context.Context, sig *escrow.SignatureCollection) error {
	_, err := r.DB.ExecContext(ctx, `
		INSERT INTO payout_signatures (task_slug, trustee_public_key_hex, l1_signature_fragment, l2_webcrypto_signature)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (task_slug, trustee_public_key_hex) DO UPDATE
		SET l1_signature_fragment = EXCLUDED.l1_signature_fragment,
		    l2_webcrypto_signature = EXCLUDED.l2_webcrypto_signature
	`, sig.TaskSlug, sig.TrusteePublicKeyHex, sig.L1SignatureFragment, sig.L2WebCryptoSignature)
	return err
}

func (r *PostgresRepository) GetSignatureCount(ctx context.Context, taskSlug string) (int, error) {
	var count int
	err := r.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM payout_signatures WHERE task_slug=$1`, taskSlug,
	).Scan(&count)
	return count, err
}

func (r *PostgresRepository) GetAllSignatures(ctx context.Context, taskSlug string) ([]escrow.SignatureCollection, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT task_slug, trustee_public_key_hex, l1_signature_fragment, l2_webcrypto_signature
		 FROM payout_signatures WHERE task_slug=$1`, taskSlug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sigs []escrow.SignatureCollection
	for rows.Next() {
		var s escrow.SignatureCollection
		if err := rows.Scan(&s.TaskSlug, &s.TrusteePublicKeyHex, &s.L1SignatureFragment, &s.L2WebCryptoSignature); err != nil {
			return nil, err
		}
		sigs = append(sigs, s)
	}
	return sigs, nil
}

func (r *PostgresRepository) SavePayoutManifest(ctx context.Context, taskSlug string, unsignedPsbtHex string, volunteerInvoice string) error {
	data, _ := json.Marshal(map[string]string{
		"unsigned_psbt_hex": unsignedPsbtHex,
		"volunteer_invoice": volunteerInvoice,
	})
	_, err := r.DB.ExecContext(ctx, `
		INSERT INTO payout_manifests (task_slug, manifest_json)
		VALUES ($1, $2)
		ON CONFLICT (task_slug) DO UPDATE SET manifest_json = EXCLUDED.manifest_json
	`, taskSlug, string(data))
	return err
}

func (r *PostgresRepository) GetPayoutManifest(ctx context.Context, taskSlug string) (string, string, error) {
	var raw string
	err := r.DB.QueryRowContext(ctx,
		`SELECT manifest_json FROM payout_manifests WHERE task_slug=$1`, taskSlug,
	).Scan(&raw)
	if err != nil {
		return "", "", err
	}
	var data map[string]string
	if err = json.Unmarshal([]byte(raw), &data); err != nil {
		return "", "", err
	}
	return data["unsigned_psbt_hex"], data["volunteer_invoice"], nil
}

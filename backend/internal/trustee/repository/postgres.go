package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"pamojabuild1/backend/internal/trustee"
)

type TrusteeRepository struct{ db *sql.DB }

func NewTrusteeRepository(db *sql.DB) *TrusteeRepository { return &TrusteeRepository{db: db} }

func (r *TrusteeRepository) Nominate(ctx context.Context, assignment *trustee.Assignment) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin trustee nomination: %w", err)
	}
	defer tx.Rollback()

	var creatorID int64
	if err := tx.QueryRowContext(ctx, `SELECT creator_id FROM tasks WHERE slug = $1 FOR UPDATE`, assignment.TaskSlug).Scan(&creatorID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return trustee.ErrTaskNotFound
		}
		return fmt.Errorf("lock trustee task: %w", err)
	}
	if creatorID != assignment.NominatedBy {
		return trustee.ErrNotTaskCreator
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO trustee_assignments (
			task_slug, trustee_index, user_id, status, nominated_by, nomination_message
		) VALUES ($1, $2, $3, 'invited', $4, $5)`,
		assignment.TaskSlug, assignment.TrusteeIndex, assignment.UserID,
		assignment.NominatedBy, assignment.NominationMessage)
	if err != nil {
		return mapWriteError(err)
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO trustee_assignment_history (
			task_slug, trustee_index, user_id, action, actor_user_id, reason
		) VALUES ($1, $2, $3, 'nominated', $4, $5)`,
		assignment.TaskSlug, assignment.TrusteeIndex, assignment.UserID,
		assignment.NominatedBy, assignment.NominationMessage)
	if err != nil {
		return fmt.Errorf("record trustee nomination: %w", err)
	}
	return tx.Commit()
}

func (r *TrusteeRepository) Accept(ctx context.Context, taskSlug string, userID int64, proofChallenge string) (*trustee.Assignment, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin trustee acceptance: %w", err)
	}
	defer tx.Rollback()

	var result trustee.Assignment
	err = tx.QueryRowContext(ctx, `
		UPDATE trustee_assignments
		SET status = 'accepted', accepted_at = CURRENT_TIMESTAMP, proof_challenge = $3
		WHERE task_slug = $1 AND user_id = $2 AND status = 'invited'
		RETURNING task_slug, trustee_index, user_id, status, nominated_by,
		          nomination_message, proof_challenge, invited_at, accepted_at`,
		taskSlug, userID, proofChallenge).Scan(
		&result.TaskSlug, &result.TrusteeIndex, &result.UserID, &result.Status,
		&result.NominatedBy, &result.NominationMessage, &result.ProofChallenge,
		&result.InvitedAt, &result.AcceptedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, trustee.ErrInvalidState
	}
	if err != nil {
		return nil, fmt.Errorf("accept trustee nomination: %w", err)
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO trustee_assignment_history
		(task_slug, trustee_index, user_id, action, actor_user_id)
		VALUES ($1, $2, $3, 'accepted', $3)`, taskSlug, result.TrusteeIndex, userID)
	if err != nil {
		return nil, fmt.Errorf("record trustee acceptance: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit trustee acceptance: %w", err)
	}
	return &result, nil
}

func (r *TrusteeRepository) Activate(ctx context.Context, registration *trustee.KeyRegistration) (*trustee.Assignment, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin trustee activation: %w", err)
	}
	defer tx.Rollback()

	var result trustee.Assignment
	err = tx.QueryRowContext(ctx, `
		UPDATE trustee_assignments
		SET status = 'active', activated_at = CURRENT_TIMESTAMP, proof_challenge = NULL
		WHERE task_slug = $1 AND user_id = $2 AND status = 'accepted'
		  AND proof_challenge = $3
		RETURNING task_slug, trustee_index, user_id, status, nominated_by,
		          nomination_message, invited_at, accepted_at, activated_at`,
		registration.TaskSlug, registration.UserID, registration.ProofChallenge).Scan(
		&result.TaskSlug, &result.TrusteeIndex, &result.UserID, &result.Status,
		&result.NominatedBy, &result.NominationMessage, &result.InvitedAt,
		&result.AcceptedAt, &result.ActivatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, trustee.ErrInvalidState
	}
	if err != nil {
		return nil, fmt.Errorf("activate trustee assignment: %w", err)
	}

	registration.TrusteeIndex = result.TrusteeIndex
	_, err = tx.ExecContext(ctx, `
		INSERT INTO trustee_key_versions (
			task_slug, trustee_index, user_id, version, xpub, web_crypto_pubkey_hex,
			xpub_proof_signature_hex, web_crypto_proof_signature_hex, proof_challenge
		) VALUES ($1, $2, $3, 1, $4, $5, $6, $7, $8)`,
		registration.TaskSlug, result.TrusteeIndex, registration.UserID,
		registration.Xpub, registration.WebCryptoPubkeyHex,
		registration.XpubProofSignatureHex, registration.WebCryptoProofSignatureHex,
		registration.ProofChallenge)
	if err != nil {
		return nil, mapWriteError(err)
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO trustee_keys (task_slug, trustee_index, user_id, xpub, web_crypto_pubkey_hex)
		VALUES ($1, $2, $3, $4, $5)`, registration.TaskSlug, result.TrusteeIndex,
		registration.UserID, registration.Xpub, registration.WebCryptoPubkeyHex)
	if err != nil {
		return nil, mapWriteError(err)
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO trustee_assignment_history
		(task_slug, trustee_index, user_id, action, actor_user_id)
		VALUES ($1, $2, $3, 'activated', $3)`, registration.TaskSlug, result.TrusteeIndex, registration.UserID)
	if err != nil {
		return nil, fmt.Errorf("record trustee activation: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit trustee activation: %w", err)
	}
	return &result, nil
}

func (r *TrusteeRepository) RotateKeys(ctx context.Context, registration *trustee.KeyRegistration, actorUserID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin trustee key rotation: %w", err)
	}
	defer tx.Rollback()
	var index int32
	err = tx.QueryRowContext(ctx, `
		SELECT trustee_index FROM trustee_assignments
		WHERE task_slug = $1 AND user_id = $2 AND status = 'active'
		  AND proof_challenge = $3 FOR UPDATE`,
		registration.TaskSlug, registration.UserID, registration.ProofChallenge).Scan(&index)
	if errors.Is(err, sql.ErrNoRows) {
		return trustee.ErrInvalidState
	}
	if err != nil {
		return fmt.Errorf("lock trustee assignment for rotation: %w", err)
	}
	var nextVersion int
	err = tx.QueryRowContext(ctx, `
		UPDATE trustee_key_versions SET revoked_at = CURRENT_TIMESTAMP,
			rotated_by = $3, rotation_reason = $4
		WHERE task_slug = $1 AND trustee_index = $2 AND revoked_at IS NULL
		RETURNING version + 1`, registration.TaskSlug, index, actorUserID, registration.RotationReason).Scan(&nextVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return trustee.ErrInvalidState
	}
	if err != nil {
		return fmt.Errorf("revoke previous trustee key: %w", err)
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO trustee_key_versions (
			task_slug, trustee_index, user_id, version, xpub, web_crypto_pubkey_hex,
			xpub_proof_signature_hex, web_crypto_proof_signature_hex, proof_challenge
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, registration.TaskSlug, index,
		registration.UserID, nextVersion, registration.Xpub, registration.WebCryptoPubkeyHex,
		registration.XpubProofSignatureHex, registration.WebCryptoProofSignatureHex,
		registration.ProofChallenge)
	if err != nil {
		return mapWriteError(err)
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE trustee_keys SET xpub=$3, web_crypto_pubkey_hex=$4
		WHERE task_slug=$1 AND trustee_index=$2`, registration.TaskSlug, index,
		registration.Xpub, registration.WebCryptoPubkeyHex)
	if err != nil {
		return fmt.Errorf("update active trustee key projection: %w", err)
	}
	_, err = tx.ExecContext(ctx, `UPDATE trustee_assignments SET proof_challenge=NULL WHERE task_slug=$1 AND trustee_index=$2`, registration.TaskSlug, index)
	if err != nil {
		return fmt.Errorf("consume trustee rotation challenge: %w", err)
	}
	return tx.Commit()
}

func (r *TrusteeRepository) IssueRotationChallenge(ctx context.Context, taskSlug string, userID int64, proofChallenge string) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE trustee_assignments SET proof_challenge=$3
		WHERE task_slug=$1 AND user_id=$2 AND status='active'`, taskSlug, userID, proofChallenge)
	if err != nil {
		return fmt.Errorf("issue trustee rotation challenge: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count trustee rotation challenge update: %w", err)
	}
	if count != 1 {
		return trustee.ErrInvalidState
	}
	return nil
}

func (r *TrusteeRepository) Replace(ctx context.Context, replacement *trustee.Replacement) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin trustee replacement: %w", err)
	}
	defer tx.Rollback()
	var creatorID, oldUserID int64
	err = tx.QueryRowContext(ctx, `SELECT creator_id FROM tasks WHERE slug=$1 FOR UPDATE`, replacement.TaskSlug).Scan(&creatorID)
	if errors.Is(err, sql.ErrNoRows) {
		return trustee.ErrTaskNotFound
	}
	if err != nil {
		return fmt.Errorf("lock trustee task for replacement: %w", err)
	}
	if creatorID != replacement.ActorUserID {
		return trustee.ErrNotTaskCreator
	}
	err = tx.QueryRowContext(ctx, `
		SELECT user_id FROM trustee_assignments
		WHERE task_slug=$1 AND trustee_index=$2 AND status IN ('invited','accepted','active') FOR UPDATE`,
		replacement.TaskSlug, replacement.TrusteeIndex).Scan(&oldUserID)
	if errors.Is(err, sql.ErrNoRows) {
		return trustee.ErrAssignmentNotFound
	}
	if err != nil {
		return fmt.Errorf("lock trustee being replaced: %w", err)
	}
	replacement.OldUserID = oldUserID
	_, err = tx.ExecContext(ctx, `DELETE FROM trustee_keys WHERE task_slug=$1 AND trustee_index=$2`, replacement.TaskSlug, replacement.TrusteeIndex)
	if err != nil {
		return fmt.Errorf("remove active trustee key projection: %w", err)
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE trustee_key_versions SET revoked_at=CURRENT_TIMESTAMP, rotated_by=$3, rotation_reason=$4
		WHERE task_slug=$1 AND trustee_index=$2 AND revoked_at IS NULL`, replacement.TaskSlug,
		replacement.TrusteeIndex, replacement.ActorUserID, replacement.Reason)
	if err != nil {
		return fmt.Errorf("revoke replaced trustee keys: %w", err)
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO trustee_assignment_history
		(task_slug, trustee_index, user_id, action, actor_user_id, reason)
		VALUES ($1,$2,$3,'replaced',$4,$5)`, replacement.TaskSlug, replacement.TrusteeIndex,
		oldUserID, replacement.ActorUserID, replacement.Reason)
	if err != nil {
		return fmt.Errorf("record trustee replacement: %w", err)
	}
	// The slot row is deliberately reused only after history and old key versions
	// are preserved. Authorization continues to read trustee_keys, so the old
	// trustee loses capabilities in the same transaction.
	_, err = tx.ExecContext(ctx, `
		UPDATE trustee_assignments SET user_id=$3, status='invited', nominated_by=$4,
			nomination_message=$5, proof_challenge=NULL, invited_at=CURRENT_TIMESTAMP,
			accepted_at=NULL, activated_at=NULL, ended_at=NULL, ended_by=NULL, end_reason=NULL
		WHERE task_slug=$1 AND trustee_index=$2`, replacement.TaskSlug,
		replacement.TrusteeIndex, replacement.NewUserID, replacement.ActorUserID, replacement.Message)
	if err != nil {
		return mapWriteError(err)
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO trustee_assignment_history
		(task_slug, trustee_index, user_id, action, actor_user_id, reason)
		VALUES ($1,$2,$3,'nominated',$4,$5)`, replacement.TaskSlug, replacement.TrusteeIndex,
		replacement.NewUserID, replacement.ActorUserID, replacement.Message)
	if err != nil {
		return fmt.Errorf("record replacement nomination: %w", err)
	}
	return tx.Commit()
}

func (r *TrusteeRepository) GetAssignmentForUser(ctx context.Context, taskSlug string, userID int64) (*trustee.Assignment, error) {
	row := r.db.QueryRowContext(ctx, assignmentSelect+` WHERE a.task_slug=$1 AND a.user_id=$2`, taskSlug, userID)
	return scanAssignment(row)
}

func (r *TrusteeRepository) GetRoster(ctx context.Context, taskSlug string) ([]trustee.Assignment, error) {
	rows, err := r.db.QueryContext(ctx, assignmentSelect+` WHERE a.task_slug=$1 ORDER BY a.trustee_index`, taskSlug)
	if err != nil {
		return nil, fmt.Errorf("list trustee roster: %w", err)
	}
	defer rows.Close()
	result := make([]trustee.Assignment, 0, 5)
	for rows.Next() {
		a, err := scanAssignment(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *a)
	}
	return result, rows.Err()
}

const assignmentSelect = `SELECT a.task_slug,a.trustee_index,a.user_id,u.display_name,a.status,
	a.nominated_by,a.nomination_message,COALESCE(a.proof_challenge,''),a.invited_at,
	a.accepted_at,a.activated_at,a.ended_at FROM trustee_assignments a JOIN users u ON u.id=a.user_id`

type scanner interface{ Scan(...any) error }

func scanAssignment(row scanner) (*trustee.Assignment, error) {
	var a trustee.Assignment
	err := row.Scan(&a.TaskSlug, &a.TrusteeIndex, &a.UserID, &a.DisplayName, &a.Status,
		&a.NominatedBy, &a.NominationMessage, &a.ProofChallenge, &a.InvitedAt,
		&a.AcceptedAt, &a.ActivatedAt, &a.EndedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, trustee.ErrAssignmentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan trustee assignment: %w", err)
	}
	return &a, nil
}

func (r *TrusteeRepository) GetKeysByTask(ctx context.Context, taskSlug string) ([]trustee.TrusteeKey, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT task_slug,trustee_index,user_id,xpub,web_crypto_pubkey_hex FROM trustee_keys WHERE task_slug=$1 ORDER BY trustee_index`, taskSlug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	keys := make([]trustee.TrusteeKey, 0, 5)
	for rows.Next() {
		var k trustee.TrusteeKey
		if err := rows.Scan(&k.TaskSlug, &k.TrusteeIndex, &k.UserID, &k.Xpub, &k.WebCryptoPubkeyHex); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (r *TrusteeRepository) GetSpecificTrustee(ctx context.Context, taskSlug string, trusteeIndex int32) (*trustee.TrusteeKey, error) {
	var k trustee.TrusteeKey
	err := r.db.QueryRowContext(ctx, `SELECT task_slug,trustee_index,user_id,xpub,web_crypto_pubkey_hex FROM trustee_keys WHERE task_slug=$1 AND trustee_index=$2`, taskSlug, trusteeIndex).Scan(&k.TaskSlug, &k.TrusteeIndex, &k.UserID, &k.Xpub, &k.WebCryptoPubkeyHex)
	if err != nil {
		return nil, err
	}
	return &k, nil
}

func mapWriteError(err error) error {
	var pg *pgconn.PgError
	if !errors.As(err, &pg) {
		return err
	}
	switch pg.Code {
	case "23503":
		return trustee.ErrTaskNotFound
	case "23505", "23514", "P0001":
		return trustee.ErrRegistrationConflict
	default:
		return err
	}
}

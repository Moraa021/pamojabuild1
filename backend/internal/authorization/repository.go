package authorization

import (
	"context"
	"database/sql"
	"fmt"
)

// Repository answers task-relationship questions used by HTTP authorization.
// These are relationships for one task, not permanent properties of an account.
type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) IsTaskTrustee(ctx context.Context, taskSlug string, userID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM trustee_keys
			WHERE task_slug = $1 AND user_id = $2
		)`,
		taskSlug, userID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check task trustee relationship: %w", err)
	}
	return exists, nil
}

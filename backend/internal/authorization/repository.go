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

func (r *Repository) HasApprovedVolunteer(ctx context.Context, taskSlug string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM task_applications
			WHERE task_slug = $1 AND status = 'approved'
		)`,
		taskSlug,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check approved volunteer relationship: %w", err)
	}
	return exists, nil
}

// ApprovedSubmissionCounts deliberately counts distinct approved applicants,
// not submission rows. Multiple evidence submissions by one volunteer must not
// make another approved volunteer appear to have completed their work.
func (r *Repository) ApprovedSubmissionCounts(ctx context.Context, taskSlug string) (approved, submitted int, err error) {
	err = r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (
				WHERE EXISTS (
					SELECT 1
					FROM task_submissions
					WHERE task_submissions.task_slug = task_applications.task_slug
					  AND task_submissions.volunteer_id = task_applications.volunteer_id
				)
			)
		FROM task_applications
		WHERE task_slug = $1 AND status = 'approved'`,
		taskSlug,
	).Scan(&approved, &submitted)
	if err != nil {
		return 0, 0, fmt.Errorf("count approved volunteer submissions: %w", err)
	}
	return approved, submitted, nil
}

func (r *Repository) IsTaskVolunteer(ctx context.Context, taskSlug string, userID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM task_applications
			WHERE task_slug = $1 AND volunteer_id = $2
		)`,
		taskSlug, userID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check task volunteer relationship: %w", err)
	}
	return exists, nil
}

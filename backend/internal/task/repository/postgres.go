package repository

import (
	"context"
	"database/sql"

	"pamojabuild1/backend/internal/task"
)

type TaskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(ctx context.Context, t *task.Task) error {
	query := `
		INSERT INTO tasks (slug, creator_id, title, description, category, region, location_detail, 
		                   status, financial_state, goal_sats, max_volunteers, volunteer_mode, image_path, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id`

	return r.db.QueryRowContext(ctx, query,
		t.Slug, t.CreatorID, t.Title, t.Description, t.Category, t.Region,
		t.LocationDetail, t.Status, t.FinancialState, t.GoalSats, t.MaxVolunteers,
		t.VolunteerMode, t.ImagePath, t.CreatedAt,
	).Scan(&t.ID)
}

func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*task.Task, error) {
	t := &task.Task{}
	query := `
		SELECT id, slug, creator_id, title, description, category, region, location_detail,
		       status, financial_state, goal_sats, max_volunteers, volunteer_mode, image_path, created_at
		FROM tasks WHERE id = $1`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&t.ID, &t.Slug, &t.CreatorID, &t.Title, &t.Description, &t.Category,
		&t.Region, &t.LocationDetail, &t.Status, &t.FinancialState, &t.GoalSats,
		&t.MaxVolunteers, &t.VolunteerMode, &t.ImagePath, &t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *TaskRepository) GetBySlug(ctx context.Context, slug string) (*task.Task, error) {
	t := &task.Task{}
	query := `
		SELECT id, slug, creator_id, title, description, category, region, location_detail,
		       status, financial_state, goal_sats, max_volunteers, volunteer_mode, image_path, created_at
		FROM tasks WHERE slug = $1`

	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&t.ID, &t.Slug, &t.CreatorID, &t.Title, &t.Description, &t.Category,
		&t.Region, &t.LocationDetail, &t.Status, &t.FinancialState, &t.GoalSats,
		&t.MaxVolunteers, &t.VolunteerMode, &t.ImagePath, &t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *TaskRepository) UpdateStatus(ctx context.Context, slug string, status string) error {
	query := `UPDATE tasks SET status = $1 WHERE slug = $2`
	_, err := r.db.ExecContext(ctx, query, status, slug)
	return err
}

func (r *TaskRepository) UpdateFinancialState(ctx context.Context, slug string, state string) error {
	query := `UPDATE tasks SET financial_state = $1 WHERE slug = $2`
	_, err := r.db.ExecContext(ctx, query, state, slug)
	return err
}

func (r *TaskRepository) List(ctx context.Context, category, region, status string) ([]task.Task, error) {
	query := `
		SELECT id, slug, creator_id, title, description, category, region, location_detail,
		       status, financial_state, goal_sats, max_volunteers, volunteer_mode, image_path, created_at
		FROM tasks WHERE 1=1`
	
	args := []interface{}{}
	argCount := 0

	if category != "" {
		argCount++
		query += ` AND category = $` + string(rune('0'+argCount))
		args = append(args, category)
	}
	if region != "" {
		argCount++
		query += ` AND region = $` + string(rune('0'+argCount))
		args = append(args, region)
	}
	if status != "" {
		argCount++
		query += ` AND status = $` + string(rune('0'+argCount))
		args = append(args, status)
	}

	query += ` ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []task.Task
	for rows.Next() {
		var t task.Task
		if err := rows.Scan(&t.ID, &t.Slug, &t.CreatorID, &t.Title, &t.Description,
			&t.Category, &t.Region, &t.LocationDetail, &t.Status, &t.FinancialState,
			&t.GoalSats, &t.MaxVolunteers, &t.VolunteerMode, &t.ImagePath, &t.CreatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}
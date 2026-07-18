package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
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

	err := r.db.QueryRowContext(ctx, query,
		t.Slug, t.CreatorID, t.Title, t.Description, t.Category, t.Region,
		t.LocationDetail, t.Status, t.FinancialState, t.GoalSats, t.MaxVolunteers,
		t.VolunteerMode, t.ImagePath, t.CreatedAt,
	).Scan(&t.ID)
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) && pgError.Code == "23505" {
		return task.ErrSlugTaken
	}
	return err
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

func (r *TaskRepository) List(ctx context.Context, options task.ListOptions) (*task.ListResult, error) {
	where, args := taskListWhere(options)

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tasks`+where, args...).Scan(&total); err != nil {
		return nil, err
	}

	query := `
		SELECT id, slug, creator_id, title, description, category, region, location_detail,
		       status, financial_state, goal_sats, max_volunteers, volunteer_mode, image_path, created_at
		FROM tasks` + where + fmt.Sprintf(
		` ORDER BY created_at DESC, id DESC LIMIT $%d OFFSET $%d`,
		len(args)+1,
		len(args)+2,
	)
	args = append(args, options.PageSize, (options.Page-1)*options.PageSize)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]task.Task, 0)
	for rows.Next() {
		var t task.Task
		if err := rows.Scan(&t.ID, &t.Slug, &t.CreatorID, &t.Title, &t.Description,
			&t.Category, &t.Region, &t.LocationDetail, &t.Status, &t.FinancialState,
			&t.GoalSats, &t.MaxVolunteers, &t.VolunteerMode, &t.ImagePath, &t.CreatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &task.ListResult{Tasks: tasks, Total: total}, nil
}

func taskListWhere(options task.ListOptions) (string, []any) {
	conditions := make([]string, 0, 3)
	args := make([]any, 0, 3)
	add := func(column, value string) {
		if value == "" {
			return
		}
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf("%s = $%d", column, len(args)))
	}
	add("category", options.Category)
	add("region", options.Region)
	add("status", options.Status)
	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

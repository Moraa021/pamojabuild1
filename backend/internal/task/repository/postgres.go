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
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin task creation: %w", err)
	}
	defer tx.Rollback()

	t.WorkStateVersion = 1
	t.FinancialStateVersion = 1
	query := `
		INSERT INTO tasks (slug, creator_id, title, description, category, region, location_detail, 
		                   status, financial_state, work_state_version, financial_state_version,
		                   goal_sats, max_volunteers, volunteer_mode, image_path, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id`

	err = tx.QueryRowContext(ctx, query,
		t.Slug, t.CreatorID, t.Title, t.Description, t.Category, t.Region,
		t.LocationDetail, t.Status, t.FinancialState, t.WorkStateVersion,
		t.FinancialStateVersion, t.GoalSats, t.MaxVolunteers, t.VolunteerMode,
		t.ImagePath, t.CreatedAt,
	).Scan(&t.ID)
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) && pgError.Code == "23505" {
		return task.ErrSlugTaken
	}
	if err != nil {
		return err
	}

	actorUserID := t.CreatorID
	initialTransitions := []task.StateTransitionCommand{
		{
			TaskSlug:       t.Slug,
			StateKind:      task.StateKindWork,
			TargetState:    t.Status,
			ActorUserID:    &actorUserID,
			Reason:         "task created",
			IdempotencyKey: "task-created-work",
		},
		{
			TaskSlug:       t.Slug,
			StateKind:      task.StateKindFinancial,
			TargetState:    t.FinancialState,
			ActorUserID:    &actorUserID,
			Reason:         "task created",
			IdempotencyKey: "task-created-financial",
		},
	}
	for _, transition := range initialTransitions {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO task_state_transitions (
				task_slug, state_kind, from_state, to_state, version,
				actor_user_id, reason, idempotency_key, created_at
			)
			VALUES ($1, $2, NULL, $3, 1, $4, $5, $6, $7)`,
			transition.TaskSlug,
			transition.StateKind,
			transition.TargetState,
			transition.ActorUserID,
			transition.Reason,
			transition.IdempotencyKey,
			t.CreatedAt,
		); err != nil {
			return fmt.Errorf("record initial %s state: %w", transition.StateKind, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit task creation: %w", err)
	}
	return nil
}

func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*task.Task, error) {
	t := &task.Task{}
	query := `
		SELECT id, slug, creator_id, title, description, category, region, location_detail,
		       status, financial_state, work_state_version, financial_state_version,
		       goal_sats, max_volunteers, volunteer_mode, image_path, created_at
		FROM tasks WHERE id = $1`

	err := scanTask(r.db.QueryRowContext(ctx, query, id), t)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *TaskRepository) GetBySlug(ctx context.Context, slug string) (*task.Task, error) {
	t := &task.Task{}
	query := `
		SELECT id, slug, creator_id, title, description, category, region, location_detail,
		       status, financial_state, work_state_version, financial_state_version,
		       goal_sats, max_volunteers, volunteer_mode, image_path, created_at
		FROM tasks WHERE slug = $1`

	err := scanTask(r.db.QueryRowContext(ctx, query, slug), t)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *TaskRepository) List(ctx context.Context, options task.ListOptions) (*task.ListResult, error) {
	where, args := taskListWhere(options)

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tasks`+where, args...).Scan(&total); err != nil {
		return nil, err
	}

	query := `
		SELECT id, slug, creator_id, title, description, category, region, location_detail,
		       status, financial_state, work_state_version, financial_state_version,
		       goal_sats, max_volunteers, volunteer_mode, image_path, created_at
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
		if err := scanTask(rows, &t); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &task.ListResult{Tasks: tasks, Total: total}, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTask(row rowScanner, t *task.Task) error {
	return row.Scan(
		&t.ID,
		&t.Slug,
		&t.CreatorID,
		&t.Title,
		&t.Description,
		&t.Category,
		&t.Region,
		&t.LocationDetail,
		&t.Status,
		&t.FinancialState,
		&t.WorkStateVersion,
		&t.FinancialStateVersion,
		&t.GoalSats,
		&t.MaxVolunteers,
		&t.VolunteerMode,
		&t.ImagePath,
		&t.CreatedAt,
	)
}

func (r *TaskRepository) ApplyStateTransitions(
	ctx context.Context,
	commands []task.StateTransitionCommand,
) (*task.StateTransitionResult, error) {
	if len(commands) == 0 {
		return nil, task.ErrStateConflict
	}
	for _, command := range commands {
		if command.TaskSlug == "" ||
			!legalTransition(command.StateKind, command.ExpectedState, command.TargetState) ||
			(command.StateKind == task.StateKindWork && command.ActorUserID == nil) {
			return nil, task.ErrStateConflict
		}
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin state transition: %w", err)
	}
	defer tx.Rollback()

	var currentWorkState, currentFinancialState string
	var workVersion, financialVersion int64
	err = tx.QueryRowContext(ctx, `
		SELECT status, financial_state, work_state_version, financial_state_version
		FROM tasks
		WHERE slug = $1
		FOR UPDATE`,
		commands[0].TaskSlug,
	).Scan(&currentWorkState, &currentFinancialState, &workVersion, &financialVersion)
	if err != nil {
		return nil, err
	}

	existing := make([]task.StateTransition, 0, len(commands))
	for _, command := range commands {
		if command.TaskSlug != commands[0].TaskSlug {
			return nil, task.ErrStateConflict
		}
		transition, err := getTransitionByIdempotencyKey(ctx, tx, command)
		if err == nil {
			existing = append(existing, *transition)
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}
	if len(existing) > 0 {
		if len(existing) != len(commands) {
			return nil, task.ErrIdempotencyConflict
		}
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit idempotent state replay: %w", err)
		}
		return &task.StateTransitionResult{Transitions: existing, Replayed: true}, nil
	}

	applied := make([]task.StateTransition, 0, len(commands))
	for _, command := range commands {
		var currentState string
		var currentVersion int64
		var updateQuery string
		switch command.StateKind {
		case task.StateKindWork:
			currentState = currentWorkState
			currentVersion = workVersion
			updateQuery = workTransitionUpdate(command)
		case task.StateKindFinancial:
			currentState = currentFinancialState
			currentVersion = financialVersion
			updateQuery = `
				UPDATE tasks
				SET financial_state = $1, financial_state_version = financial_state_version + 1
				WHERE slug = $2 AND financial_state = $3 AND financial_state_version = $4`
		default:
			return nil, task.ErrStateConflict
		}
		if currentState != command.ExpectedState || currentState == command.TargetState {
			return nil, task.ErrStateConflict
		}

		updateArguments := []any{
			command.TargetState,
			command.TaskSlug,
			command.ExpectedState,
			currentVersion,
		}
		if command.StateKind == task.StateKindWork {
			updateArguments = append(updateArguments, *command.ActorUserID)
		}
		result, err := tx.ExecContext(ctx, updateQuery, updateArguments...)
		if err != nil {
			return nil, err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return nil, err
		}
		if affected != 1 {
			return nil, task.ErrStateConflict
		}

		fromState := currentState
		transition := task.StateTransition{
			TaskSlug:       command.TaskSlug,
			StateKind:      command.StateKind,
			FromState:      &fromState,
			ToState:        command.TargetState,
			Version:        currentVersion + 1,
			ActorUserID:    command.ActorUserID,
			Reason:         command.Reason,
			IdempotencyKey: command.IdempotencyKey,
		}
		err = tx.QueryRowContext(ctx, `
			INSERT INTO task_state_transitions (
				task_slug, state_kind, from_state, to_state, version,
				actor_user_id, reason, idempotency_key
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id, created_at`,
			transition.TaskSlug,
			transition.StateKind,
			transition.FromState,
			transition.ToState,
			transition.Version,
			transition.ActorUserID,
			transition.Reason,
			transition.IdempotencyKey,
		).Scan(&transition.ID, &transition.CreatedAt)
		if err != nil {
			var pgError *pgconn.PgError
			if errors.As(err, &pgError) && pgError.Code == "23505" {
				return nil, task.ErrIdempotencyConflict
			}
			return nil, err
		}
		applied = append(applied, transition)

		if command.StateKind == task.StateKindWork {
			currentWorkState = command.TargetState
			workVersion = transition.Version
		} else {
			currentFinancialState = command.TargetState
			financialVersion = transition.Version
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit state transition: %w", err)
	}
	return &task.StateTransitionResult{Transitions: applied}, nil
}

func legalTransition(stateKind, fromState, toState string) bool {
	switch stateKind {
	case task.StateKindWork:
		return (fromState == task.WorkStateOpen && toState == task.WorkStateInProgress) ||
			(fromState == task.WorkStateInProgress && toState == task.WorkStatePendingVerification) ||
			(fromState == task.WorkStatePendingVerification && toState == task.WorkStateCompleted)
	case task.StateKindFinancial:
		return (fromState == task.FinancialStateActive && toState == task.FinancialStateLiquidating) ||
			(fromState == task.FinancialStateLiquidating && toState == task.FinancialStateReadyForPayout) ||
			(fromState == task.FinancialStateReadyForPayout && toState == task.FinancialStatePayoutProcessing) ||
			(fromState == task.FinancialStatePayoutProcessing && toState == task.FinancialStateArchived)
	default:
		return false
	}
}

func workTransitionUpdate(command task.StateTransitionCommand) string {
	base := `
		UPDATE tasks
		SET status = $1, work_state_version = work_state_version + 1
		WHERE slug = $2 AND status = $3 AND work_state_version = $4`
	switch {
	case command.ExpectedState == task.WorkStateOpen &&
		command.TargetState == task.WorkStateInProgress:
		// Re-evaluate volunteer approval in the conditional write instead of
		// trusting the earlier service check. Step 6 relationship mutations must
		// take the same task-row lock to serialize changes across this boundary.
		return base + `
			AND creator_id = $5
			AND EXISTS (
				SELECT 1 FROM task_applications
				WHERE task_slug = tasks.slug AND status = 'approved'
			)`
	case command.ExpectedState == task.WorkStateInProgress &&
		command.TargetState == task.WorkStatePendingVerification:
		return base + `
			AND creator_id = $5
			AND EXISTS (
				SELECT 1 FROM task_applications
				WHERE task_slug = tasks.slug AND status = 'approved'
			)
			AND NOT EXISTS (
				SELECT 1
				FROM task_applications
				WHERE task_slug = tasks.slug
				  AND status = 'approved'
				  AND NOT EXISTS (
					SELECT 1
					FROM task_submissions
					WHERE task_submissions.task_slug = task_applications.task_slug
					  AND task_submissions.volunteer_id = task_applications.volunteer_id
				  )
			)`
	default:
		// Verification authority is checked again inside the write transaction.
		// A service-level check alone could race trustee removal.
		return base + `
			AND creator_id <> $5
			AND EXISTS (
				SELECT 1 FROM trustee_keys
				WHERE task_slug = tasks.slug AND user_id = $5
			)
			AND NOT EXISTS (
				SELECT 1 FROM task_applications
				WHERE task_slug = tasks.slug AND volunteer_id = $5
			)`
	}
}

func getTransitionByIdempotencyKey(
	ctx context.Context,
	tx *sql.Tx,
	command task.StateTransitionCommand,
) (*task.StateTransition, error) {
	transition := &task.StateTransition{}
	var fromState sql.NullString
	var actorUserID sql.NullInt64
	err := tx.QueryRowContext(ctx, `
		SELECT id, task_slug, state_kind, from_state, to_state, version,
		       actor_user_id, reason, idempotency_key, created_at
		FROM task_state_transitions
		WHERE task_slug = $1 AND state_kind = $2 AND idempotency_key = $3`,
		command.TaskSlug,
		command.StateKind,
		command.IdempotencyKey,
	).Scan(
		&transition.ID,
		&transition.TaskSlug,
		&transition.StateKind,
		&fromState,
		&transition.ToState,
		&transition.Version,
		&actorUserID,
		&transition.Reason,
		&transition.IdempotencyKey,
		&transition.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if fromState.Valid {
		transition.FromState = &fromState.String
	}
	if actorUserID.Valid {
		transition.ActorUserID = &actorUserID.Int64
	}
	if transition.ToState != command.TargetState ||
		transition.Reason != command.Reason ||
		!sameActor(transition.ActorUserID, command.ActorUserID) ||
		transition.FromState == nil ||
		*transition.FromState != command.ExpectedState {
		return nil, task.ErrIdempotencyConflict
	}
	return transition, nil
}

func sameActor(left, right *int64) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func (r *TaskRepository) ListStateTransitions(
	ctx context.Context,
	taskSlug string,
	options task.StateHistoryOptions,
) (*task.StateHistoryResult, error) {
	where := ` WHERE task_slug = $1`
	args := []any{taskSlug}
	if options.StateKind != "" {
		args = append(args, options.StateKind)
		where += fmt.Sprintf(` AND state_kind = $%d`, len(args))
	}

	var total int
	if err := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM task_state_transitions`+where,
		args...,
	).Scan(&total); err != nil {
		return nil, err
	}

	query := `
		SELECT id, task_slug, state_kind, from_state, to_state, version,
		       actor_user_id, reason, idempotency_key, created_at
		FROM task_state_transitions` + where + fmt.Sprintf(
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

	transitions := make([]task.StateTransition, 0)
	for rows.Next() {
		var transition task.StateTransition
		var fromState sql.NullString
		var actorUserID sql.NullInt64
		if err := rows.Scan(
			&transition.ID,
			&transition.TaskSlug,
			&transition.StateKind,
			&fromState,
			&transition.ToState,
			&transition.Version,
			&actorUserID,
			&transition.Reason,
			&transition.IdempotencyKey,
			&transition.CreatedAt,
		); err != nil {
			return nil, err
		}
		if fromState.Valid {
			transition.FromState = &fromState.String
		}
		if actorUserID.Valid {
			transition.ActorUserID = &actorUserID.Int64
		}
		transitions = append(transitions, transition)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &task.StateHistoryResult{Transitions: transitions, Total: total}, nil
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

package repository

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"pamojabuild1/backend/internal/task"
	"pamojabuild1/backend/internal/testsupport"
)

func TestCreateAndTransitionTaskState(t *testing.T) {
	database := testsupport.NewPostgresDatabase(t)
	repository := NewTaskRepository(database)
	creatorID := insertUser(t, database, "+254700001001")
	created := testTask(creatorID, "repository-state-task")

	if err := repository.Create(context.Background(), created); err != nil {
		t.Fatalf("create task: %v", err)
	}
	volunteerID := insertUser(t, database, "+254700001011")
	insertApprovedApplication(t, database, created.Slug, volunteerID)
	if created.WorkStateVersion != 1 || created.FinancialStateVersion != 1 {
		t.Fatalf("unexpected initial versions: %#v", created)
	}

	command := task.StateTransitionCommand{
		TaskSlug:       created.Slug,
		StateKind:      task.StateKindWork,
		ExpectedState:  task.WorkStateOpen,
		TargetState:    task.WorkStateInProgress,
		ActorUserID:    &creatorID,
		Reason:         "work is ready",
		IdempotencyKey: "start-repository-test",
	}
	result, err := repository.ApplyStateTransitions(context.Background(), []task.StateTransitionCommand{command})
	if err != nil {
		t.Fatalf("apply transition: %v", err)
	}
	if result.Replayed || len(result.Transitions) != 1 || result.Transitions[0].Version != 2 {
		t.Fatalf("unexpected transition result: %#v", result)
	}

	replay, err := repository.ApplyStateTransitions(context.Background(), []task.StateTransitionCommand{command})
	if err != nil {
		t.Fatalf("replay transition: %v", err)
	}
	if !replay.Replayed || replay.Transitions[0].ID != result.Transitions[0].ID {
		t.Fatalf("expected exact idempotent replay: %#v", replay)
	}

	found, err := repository.GetBySlug(context.Background(), created.Slug)
	if err != nil {
		t.Fatalf("get transitioned task: %v", err)
	}
	if found.Status != task.WorkStateInProgress || found.WorkStateVersion != 2 {
		t.Fatalf("transition was not persisted: %#v", found)
	}

	history, err := repository.ListStateTransitions(context.Background(), created.Slug, task.StateHistoryOptions{
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("list state history: %v", err)
	}
	if history.Total != 3 || len(history.Transitions) != 3 {
		t.Fatalf("expected two baselines and one transition, got %#v", history)
	}
}

func TestConcurrentTransitionsAllowOneWinner(t *testing.T) {
	database := testsupport.NewPostgresDatabase(t)
	repository := NewTaskRepository(database)
	creatorID := insertUser(t, database, "+254700001002")
	created := testTask(creatorID, "concurrent-state-task")
	if err := repository.Create(context.Background(), created); err != nil {
		t.Fatalf("create task: %v", err)
	}
	volunteerID := insertUser(t, database, "+254700001012")
	insertApprovedApplication(t, database, created.Slug, volunteerID)

	start := make(chan struct{})
	results := make(chan error, 2)
	var workers sync.WaitGroup
	for _, key := range []string{"concurrent-a", "concurrent-b"} {
		workers.Add(1)
		go func(idempotencyKey string) {
			defer workers.Done()
			<-start
			_, err := repository.ApplyStateTransitions(context.Background(), []task.StateTransitionCommand{{
				TaskSlug:       created.Slug,
				StateKind:      task.StateKindWork,
				ExpectedState:  task.WorkStateOpen,
				TargetState:    task.WorkStateInProgress,
				ActorUserID:    &creatorID,
				Reason:         "concurrent start",
				IdempotencyKey: idempotencyKey,
			}})
			results <- err
		}(key)
	}
	close(start)
	workers.Wait()
	close(results)

	var succeeded, conflicted int
	for err := range results {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, task.ErrStateConflict):
			conflicted++
		default:
			t.Fatalf("unexpected concurrent result: %v", err)
		}
	}
	if succeeded != 1 || conflicted != 1 {
		t.Fatalf("expected one winner and one conflict, got success=%d conflict=%d", succeeded, conflicted)
	}
}

func TestLegalTransitionGraphRejectsStateJumps(t *testing.T) {
	if legalTransition(task.StateKindWork, task.WorkStateOpen, task.WorkStateCompleted) {
		t.Fatal("work state jump must be rejected")
	}
	if legalTransition(task.StateKindFinancial, task.FinancialStateActive, task.FinancialStateArchived) {
		t.Fatal("financial state jump must be rejected")
	}
	if !legalTransition(task.StateKindWork, task.WorkStateOpen, task.WorkStateInProgress) {
		t.Fatal("expected linear work transition to be allowed")
	}
}

func insertUser(t *testing.T, database *sql.DB, phone string) int64 {
	t.Helper()
	var userID int64
	if err := database.QueryRow(`
		INSERT INTO users (phone_number, password_hash, display_name)
		VALUES ($1, 'test-password-hash', 'State Test')
		RETURNING id`,
		phone,
	).Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return userID
}

func insertApprovedApplication(t *testing.T, database *sql.DB, taskSlug string, volunteerID int64) {
	t.Helper()
	if _, err := database.Exec(`
		INSERT INTO task_applications (task_slug, volunteer_id, status)
		VALUES ($1, $2, 'approved')`,
		taskSlug,
		volunteerID,
	); err != nil {
		t.Fatalf("insert approved application: %v", err)
	}
}

func testTask(creatorID int64, slug string) *task.Task {
	return &task.Task{
		Slug:           slug,
		CreatorID:      creatorID,
		Title:          "State Test",
		Description:    "Exercise state persistence",
		Category:       "community",
		Region:         "Nairobi",
		Status:         task.WorkStateOpen,
		FinancialState: task.FinancialStateActive,
		VolunteerMode:  "approval_required",
		CreatedAt:      time.Now().UTC(),
	}
}

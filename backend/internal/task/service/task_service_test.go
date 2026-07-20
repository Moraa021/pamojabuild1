package service

import (
	"context"
	"errors"
	"testing"

	"pamojabuild1/backend/internal/events"
	"pamojabuild1/backend/internal/task"
)

type mockTaskRepo struct {
	created *task.Task
	err     error
	getErr  error
	found   *task.Task
	applied []task.StateTransitionCommand
	result  *task.StateTransitionResult
}

func (m *mockTaskRepo) Create(ctx context.Context, t *task.Task) error {
	if m.err != nil {
		return m.err
	}
	m.created = t
	return nil
}
func (m *mockTaskRepo) GetByID(ctx context.Context, id int64) (*task.Task, error) { return nil, nil }
func (m *mockTaskRepo) GetBySlug(ctx context.Context, slug string) (*task.Task, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.found != nil {
		return m.found, nil
	}
	return &task.Task{Slug: slug}, nil
}
func (m *mockTaskRepo) List(ctx context.Context, options task.ListOptions) (*task.ListResult, error) {
	return &task.ListResult{Tasks: []task.Task{}}, nil
}
func (m *mockTaskRepo) ApplyStateTransitions(_ context.Context, commands []task.StateTransitionCommand) (*task.StateTransitionResult, error) {
	m.applied = append([]task.StateTransitionCommand(nil), commands...)
	if m.err != nil {
		return nil, m.err
	}
	if m.result != nil {
		return m.result, nil
	}
	return &task.StateTransitionResult{}, nil
}
func (m *mockTaskRepo) ListStateTransitions(context.Context, string, task.StateHistoryOptions) (*task.StateHistoryResult, error) {
	return &task.StateHistoryResult{}, nil
}

type mockRelationships struct {
	hasApproved bool
	approved    int
	submitted   int
	trustee     bool
	volunteer   bool
	err         error
}

func (m mockRelationships) HasApprovedVolunteer(context.Context, string) (bool, error) {
	return m.hasApproved, m.err
}
func (m mockRelationships) ApprovedSubmissionCounts(context.Context, string) (int, int, error) {
	return m.approved, m.submitted, m.err
}
func (m mockRelationships) IsTaskTrustee(context.Context, string, int64) (bool, error) {
	return m.trustee, m.err
}
func (m mockRelationships) IsTaskVolunteer(context.Context, string, int64) (bool, error) {
	return m.volunteer, m.err
}

func TestCreateCampaignPublishesEvent(t *testing.T) {
	repo := &mockTaskRepo{}
	bus := events.NewEventBus()
	var got bool
	unsub := bus.Subscribe(events.TaskCreated, func(e events.Event) {
		if payload, ok := e.Payload.(events.TaskCreatedPayload); ok && payload.Title == "Hello" {
			got = true
		}
	})
	defer unsub()

	svc := NewTaskService(repo, mockRelationships{}, bus)
	ctx := context.Background()
	tsk := &task.Task{
		CreatorID:     1,
		Title:         "Hello",
		Description:   "Useful work",
		Category:      "community",
		Region:        "Nairobi",
		VolunteerMode: "open",
	}
	res, err := svc.CreateCampaign(ctx, tsk)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Slug == "" {
		t.Fatalf("expected slug set")
	}
	// allow goroutine to run
	// small sleep
	// but avoid import time by checking repo
	if repo.created == nil {
		t.Fatalf("expected repo.Create called")
	}
	// Replay history to ensure event published
	bus.Replay()
	if !got {
		t.Fatalf("expected TaskCreated event to be published")
	}
}

func TestCreateCampaignRepoError(t *testing.T) {
	repo := &mockTaskRepo{err: errors.New("oops")}
	bus := events.NewEventBus()
	svc := NewTaskService(repo, mockRelationships{}, bus)
	_, err := svc.CreateCampaign(context.Background(), &task.Task{
		CreatorID:     1,
		Title:         "X",
		Description:   "Useful work",
		Category:      "community",
		Region:        "Nairobi",
		VolunteerMode: "open",
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestCreateCampaignValidatesAtServiceBoundary(t *testing.T) {
	repo := &mockTaskRepo{}
	svc := NewTaskService(repo, mockRelationships{}, nil)

	_, err := svc.CreateCampaign(context.Background(), &task.Task{CreatorID: 1, Title: "Only a title"})
	if !errors.Is(err, ErrInvalidTask) {
		t.Fatalf("expected ErrInvalidTask, got %v", err)
	}
	if repo.created != nil {
		t.Fatal("invalid task must not reach the repository")
	}
}

func TestListTasksRejectsOutOfRangePageSize(t *testing.T) {
	svc := NewTaskService(&mockTaskRepo{}, mockRelationships{}, nil)

	_, err := svc.ListTasks(context.Background(), task.ListOptions{Page: 1, PageSize: 101})
	if !errors.Is(err, ErrInvalidTaskFilter) {
		t.Fatalf("expected ErrInvalidTaskFilter, got %v", err)
	}
}

func TestStartTaskRequiresCreatorAndApprovedVolunteer(t *testing.T) {
	repo := &mockTaskRepo{found: &task.Task{
		Slug:      "community-garden",
		CreatorID: 10,
		Status:    task.WorkStateOpen,
	}}
	svc := NewTaskService(repo, mockRelationships{}, nil)

	if _, err := svc.StartTask(context.Background(), "community-garden", 11, "", "start-1"); !errors.Is(err, ErrStateActionForbidden) {
		t.Fatalf("expected creator authorization failure, got %v", err)
	}
	if _, err := svc.StartTask(context.Background(), "community-garden", 10, "", "start-1"); !errors.Is(err, ErrNoApprovedVolunteers) {
		t.Fatalf("expected approved volunteer readiness failure, got %v", err)
	}

	svc = NewTaskService(repo, mockRelationships{hasApproved: true}, nil)
	if _, err := svc.StartTask(context.Background(), "community-garden", 10, "", "start-1"); err != nil {
		t.Fatalf("start ready task: %v", err)
	}
	if len(repo.applied) != 1 ||
		repo.applied[0].ExpectedState != task.WorkStateOpen ||
		repo.applied[0].TargetState != task.WorkStateInProgress ||
		repo.applied[0].ActorUserID == nil ||
		*repo.applied[0].ActorUserID != 10 {
		t.Fatalf("unexpected transition command: %#v", repo.applied)
	}
}

func TestSubmitForVerificationRequiresEveryApprovedVolunteerSubmission(t *testing.T) {
	repo := &mockTaskRepo{found: &task.Task{
		Slug:      "community-garden",
		CreatorID: 10,
		Status:    task.WorkStateInProgress,
	}}
	svc := NewTaskService(repo, mockRelationships{approved: 2, submitted: 1}, nil)
	if _, err := svc.SubmitForVerification(context.Background(), "community-garden", 10, "", "submit-1"); !errors.Is(err, ErrSubmissionsIncomplete) {
		t.Fatalf("expected incomplete submissions, got %v", err)
	}

	svc = NewTaskService(repo, mockRelationships{approved: 2, submitted: 2}, nil)
	if _, err := svc.SubmitForVerification(context.Background(), "community-garden", 10, "", "submit-1"); err != nil {
		t.Fatalf("submit completed work: %v", err)
	}
	if len(repo.applied) != 1 ||
		repo.applied[0].TargetState != task.WorkStatePendingVerification {
		t.Fatalf("unexpected transition command: %#v", repo.applied)
	}
}

func TestVerifyTaskRequiresIndependentTrusteeAndTransitionsAtomically(t *testing.T) {
	repo := &mockTaskRepo{found: &task.Task{
		Slug:           "community-garden",
		CreatorID:      10,
		Status:         task.WorkStatePendingVerification,
		FinancialState: task.FinancialStateActive,
	}}
	svc := NewTaskService(repo, mockRelationships{trustee: true}, nil)
	if _, err := svc.VerifyTask(context.Background(), "community-garden", 10, "", "verify-1"); !errors.Is(err, ErrIndependentVerifier) {
		t.Fatalf("creator must not verify own task, got %v", err)
	}

	svc = NewTaskService(repo, mockRelationships{trustee: true, volunteer: true}, nil)
	if _, err := svc.VerifyTask(context.Background(), "community-garden", 20, "", "verify-1"); !errors.Is(err, ErrIndependentVerifier) {
		t.Fatalf("volunteer must not verify own task, got %v", err)
	}

	svc = NewTaskService(repo, mockRelationships{trustee: true}, nil)
	if _, err := svc.VerifyTask(context.Background(), "community-garden", 20, "", "verify-1"); err != nil {
		t.Fatalf("verify task: %v", err)
	}
	if len(repo.applied) != 2 {
		t.Fatalf("work and financial transitions must be one repository operation: %#v", repo.applied)
	}
	if repo.applied[0].TargetState != task.WorkStateCompleted ||
		repo.applied[1].TargetState != task.FinancialStateLiquidating ||
		repo.applied[0].IdempotencyKey != "verify-1:work" ||
		repo.applied[1].IdempotencyKey != "verify-1:financial" {
		t.Fatalf("unexpected verification transitions: %#v", repo.applied)
	}
}

func TestFinancialStateRejectsClientStyleLockdownJump(t *testing.T) {
	repo := &mockTaskRepo{found: &task.Task{Slug: "community-garden"}}
	svc := NewTaskService(repo, mockRelationships{}, nil)

	_, err := svc.AdvanceFinancialState(
		context.Background(),
		"community-garden",
		task.FinancialStateSystemLockdown,
		"manual jump",
		"lockdown-1",
	)
	if !errors.Is(err, ErrInvalidStateAction) {
		t.Fatalf("expected unsupported recovery transition rejection, got %v", err)
	}
	if len(repo.applied) != 0 {
		t.Fatal("invalid financial transition must not reach repository")
	}
}

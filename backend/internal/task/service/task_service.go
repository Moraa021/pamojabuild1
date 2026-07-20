package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gosimple/slug"

	"pamojabuild1/backend/internal/events"
	"pamojabuild1/backend/internal/task"
)

var (
	ErrTaskNotFound          = errors.New("task not found")
	ErrTaskConflict          = errors.New("a task with this generated slug already exists")
	ErrInvalidTask           = errors.New("invalid task")
	ErrInvalidTaskFilter     = errors.New("invalid task list filters")
	ErrInvalidStateAction    = errors.New("invalid task state action")
	ErrStateConflict         = errors.New("task is not in the required state")
	ErrStateActionForbidden  = errors.New("account is not allowed to transition this task")
	ErrNoApprovedVolunteers  = errors.New("at least one approved volunteer is required")
	ErrSubmissionsIncomplete = errors.New("all approved volunteers must submit work first")
	ErrIndependentVerifier   = errors.New("verification requires an independent task trustee")
	ErrInvalidStateHistory   = errors.New("invalid task state history filters")
)

var idempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]+$`)

type TaskService struct {
	repo          task.Repository
	relationships task.RelationshipReader
	eventBus      *events.EventBus
}

func NewTaskService(
	repo task.Repository,
	relationships task.RelationshipReader,
	eventBus *events.EventBus,
) *TaskService {
	return &TaskService{repo: repo, relationships: relationships, eventBus: eventBus}
}

func (s *TaskService) CreateCampaign(ctx context.Context, t *task.Task) (*task.Task, error) {
	if err := validateTask(t); err != nil {
		return nil, err
	}
	t.Slug = slug.Make(t.Title)
	if t.Slug == "" {
		return nil, fmt.Errorf("%w: title must contain letters or numbers", ErrInvalidTask)
	}
	t.Status = task.WorkStateOpen
	t.FinancialState = task.FinancialStateActive
	t.CreatedAt = time.Now().UTC()

	if err := s.repo.Create(ctx, t); err != nil {
		if errors.Is(err, task.ErrSlugTaken) {
			return nil, ErrTaskConflict
		}
		return nil, fmt.Errorf("create task: %w", err)
	}

	if s.eventBus != nil {
		s.eventBus.Publish(events.Event{
			Type: events.TaskCreated,
			Payload: events.TaskCreatedPayload{
				TaskSlug:      t.Slug,
				CreatorUserID: t.CreatorID,
				Title:         t.Title,
				Category:      t.Category,
				Region:        t.Region,
				GoalSats:      t.GoalSats,
			},
		})
	}
	return t, nil
}

func (s *TaskService) GetTask(ctx context.Context, taskSlug string) (*task.Task, error) {
	taskSlug = strings.TrimSpace(taskSlug)
	if taskSlug == "" || len(taskSlug) > 255 {
		return nil, ErrTaskNotFound
	}
	found, err := s.repo.GetBySlug(ctx, taskSlug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("get task: %w", err)
	}
	return found, nil
}

func (s *TaskService) ListTasks(ctx context.Context, options task.ListOptions) (*task.ListResult, error) {
	options.Category = strings.TrimSpace(options.Category)
	options.Region = strings.TrimSpace(options.Region)
	options.Status = strings.TrimSpace(options.Status)
	if options.Page < 1 || options.PageSize < 1 || options.PageSize > 100 ||
		len(options.Category) > 100 || len(options.Region) > 100 || len(options.Status) > 50 {
		return nil, ErrInvalidTaskFilter
	}
	if options.Status != "" && !isWorkState(options.Status) {
		return nil, ErrInvalidTaskFilter
	}
	return s.repo.List(ctx, options)
}

func (s *TaskService) StartTask(
	ctx context.Context,
	taskSlug string,
	actorUserID int64,
	reason string,
	idempotencyKey string,
) (*task.StateTransitionResult, error) {
	reason, err := validateStateAction(actorUserID, reason, idempotencyKey, 128)
	if err != nil {
		return nil, err
	}
	current, err := s.GetTask(ctx, taskSlug)
	if err != nil {
		return nil, err
	}
	if current.CreatorID != actorUserID {
		return nil, ErrStateActionForbidden
	}
	if current.Status == task.WorkStateOpen {
		ready, err := s.relationships.HasApprovedVolunteer(ctx, taskSlug)
		if err != nil {
			return nil, fmt.Errorf("check approved volunteers: %w", err)
		}
		if !ready {
			return nil, ErrNoApprovedVolunteers
		}
	}

	result, err := s.repo.ApplyStateTransitions(ctx, []task.StateTransitionCommand{{
		TaskSlug:       taskSlug,
		StateKind:      task.StateKindWork,
		ExpectedState:  task.WorkStateOpen,
		TargetState:    task.WorkStateInProgress,
		ActorUserID:    actorPointer(actorUserID),
		Reason:         defaultReason(reason, "creator started task work"),
		IdempotencyKey: idempotencyKey,
	}})
	if err != nil {
		return nil, mapTransitionError(err)
	}
	s.publishTransitions(result)
	return result, nil
}

func (s *TaskService) SubmitForVerification(
	ctx context.Context,
	taskSlug string,
	actorUserID int64,
	reason string,
	idempotencyKey string,
) (*task.StateTransitionResult, error) {
	reason, err := validateStateAction(actorUserID, reason, idempotencyKey, 128)
	if err != nil {
		return nil, err
	}
	current, err := s.GetTask(ctx, taskSlug)
	if err != nil {
		return nil, err
	}
	if current.CreatorID != actorUserID {
		return nil, ErrStateActionForbidden
	}
	if current.Status == task.WorkStateInProgress {
		approved, submitted, err := s.relationships.ApprovedSubmissionCounts(ctx, taskSlug)
		if err != nil {
			return nil, fmt.Errorf("check submission readiness: %w", err)
		}
		if approved == 0 {
			return nil, ErrNoApprovedVolunteers
		}
		if submitted < approved {
			return nil, ErrSubmissionsIncomplete
		}
	}

	result, err := s.repo.ApplyStateTransitions(ctx, []task.StateTransitionCommand{{
		TaskSlug:       taskSlug,
		StateKind:      task.StateKindWork,
		ExpectedState:  task.WorkStateInProgress,
		TargetState:    task.WorkStatePendingVerification,
		ActorUserID:    actorPointer(actorUserID),
		Reason:         defaultReason(reason, "creator submitted completed work for independent verification"),
		IdempotencyKey: idempotencyKey,
	}})
	if err != nil {
		return nil, mapTransitionError(err)
	}
	s.publishTransitions(result)
	return result, nil
}

func (s *TaskService) VerifyTask(
	ctx context.Context,
	taskSlug string,
	actorUserID int64,
	reason string,
	idempotencyKey string,
) (*task.StateTransitionResult, error) {
	reason, err := validateStateAction(actorUserID, reason, idempotencyKey, 118)
	if err != nil {
		return nil, err
	}
	current, err := s.GetTask(ctx, taskSlug)
	if err != nil {
		return nil, err
	}
	if current.CreatorID == actorUserID {
		return nil, ErrIndependentVerifier
	}
	isVolunteer, err := s.relationships.IsTaskVolunteer(ctx, taskSlug, actorUserID)
	if err != nil {
		return nil, fmt.Errorf("check volunteer relationship: %w", err)
	}
	if isVolunteer {
		return nil, ErrIndependentVerifier
	}
	isTrustee, err := s.relationships.IsTaskTrustee(ctx, taskSlug, actorUserID)
	if err != nil {
		return nil, fmt.Errorf("check trustee relationship: %w", err)
	}
	if !isTrustee {
		return nil, ErrIndependentVerifier
	}

	actor := actorPointer(actorUserID)
	reason = defaultReason(reason, "independent trustee verified task work")
	result, err := s.repo.ApplyStateTransitions(ctx, []task.StateTransitionCommand{
		{
			TaskSlug:       taskSlug,
			StateKind:      task.StateKindWork,
			ExpectedState:  task.WorkStatePendingVerification,
			TargetState:    task.WorkStateCompleted,
			ActorUserID:    actor,
			Reason:         reason,
			IdempotencyKey: idempotencyKey + ":work",
		},
		{
			TaskSlug:       taskSlug,
			StateKind:      task.StateKindFinancial,
			ExpectedState:  task.FinancialStateActive,
			TargetState:    task.FinancialStateLiquidating,
			ActorUserID:    actor,
			Reason:         "verified work completed; donations are now closed",
			IdempotencyKey: idempotencyKey + ":financial",
		},
	})
	if err != nil {
		return nil, mapTransitionError(err)
	}
	s.publishTransitions(result)
	return result, nil
}

// AdvanceFinancialState is an internal boundary for later payout services. It
// intentionally has no public task route: browser callers must never choose a
// money-moving lifecycle state.
func (s *TaskService) AdvanceFinancialState(
	ctx context.Context,
	taskSlug string,
	targetState string,
	reason string,
	idempotencyKey string,
) (*task.StateTransitionResult, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" || len(reason) > 500 ||
		!validIdempotencyKey(idempotencyKey, 128) {
		return nil, ErrInvalidStateAction
	}
	expectedStates := map[string]string{
		task.FinancialStateLiquidating:      task.FinancialStateActive,
		task.FinancialStateReadyForPayout:   task.FinancialStateLiquidating,
		task.FinancialStatePayoutProcessing: task.FinancialStateReadyForPayout,
		task.FinancialStateArchived:         task.FinancialStatePayoutProcessing,
	}
	expectedState, ok := expectedStates[targetState]
	if !ok {
		// SYSTEM_LOCKDOWN needs an approved recovery policy before it can become
		// an executable transition rather than an irreversible string.
		return nil, ErrInvalidStateAction
	}
	if _, err := s.GetTask(ctx, taskSlug); err != nil {
		return nil, err
	}
	result, err := s.repo.ApplyStateTransitions(ctx, []task.StateTransitionCommand{{
		TaskSlug:       taskSlug,
		StateKind:      task.StateKindFinancial,
		ExpectedState:  expectedState,
		TargetState:    targetState,
		Reason:         reason,
		IdempotencyKey: idempotencyKey,
	}})
	if err != nil {
		return nil, mapTransitionError(err)
	}
	s.publishTransitions(result)
	return result, nil
}

func (s *TaskService) ListStateTransitions(
	ctx context.Context,
	taskSlug string,
	options task.StateHistoryOptions,
) (*task.StateHistoryResult, error) {
	options.StateKind = strings.TrimSpace(options.StateKind)
	if options.Page < 1 || options.PageSize < 1 || options.PageSize > 100 ||
		(options.StateKind != "" &&
			options.StateKind != task.StateKindWork &&
			options.StateKind != task.StateKindFinancial) {
		return nil, ErrInvalidStateHistory
	}
	if _, err := s.GetTask(ctx, taskSlug); err != nil {
		return nil, err
	}
	result, err := s.repo.ListStateTransitions(ctx, taskSlug, options)
	if err != nil {
		return nil, fmt.Errorf("list task state transitions: %w", err)
	}
	return result, nil
}

func (s *TaskService) publishTransitions(result *task.StateTransitionResult) {
	if result == nil || result.Replayed || s.eventBus == nil {
		return
	}
	for _, transition := range result.Transitions {
		switch transition.StateKind {
		case task.StateKindWork:
			s.eventBus.Publish(events.Event{
				Type: events.TaskStatusChanged,
				Payload: events.TaskStatusChangedPayload{
					TaskSlug:  transition.TaskSlug,
					OldStatus: valueOrEmpty(transition.FromState),
					NewStatus: transition.ToState,
					Version:   transition.Version,
				},
			})
		case task.StateKindFinancial:
			s.eventBus.Publish(events.Event{
				Type: events.FinancialStateChanged,
				Payload: events.FinancialStateChangedPayload{
					TaskSlug: transition.TaskSlug,
					OldState: valueOrEmpty(transition.FromState),
					NewState: transition.ToState,
					Version:  transition.Version,
				},
			})
			if transition.ToState == task.FinancialStateLiquidating {
				s.eventBus.Publish(events.Event{
					Type: events.TaskLiquidating,
					Payload: events.TaskLiquidatingPayload{
						TaskSlug: transition.TaskSlug,
						Reason:   transition.Reason,
					},
				})
			}
		}
	}
}

func mapTransitionError(err error) error {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return ErrTaskNotFound
	case errors.Is(err, task.ErrStateConflict),
		errors.Is(err, task.ErrIdempotencyConflict):
		return ErrStateConflict
	default:
		return fmt.Errorf("apply task state transition: %w", err)
	}
}

func validateStateAction(
	actorUserID int64,
	reason string,
	idempotencyKey string,
	maxKeyLength int,
) (string, error) {
	reason = strings.TrimSpace(reason)
	if actorUserID <= 0 || len(reason) > 500 ||
		!validIdempotencyKey(idempotencyKey, maxKeyLength) {
		return "", ErrInvalidStateAction
	}
	return reason, nil
}

func validIdempotencyKey(value string, maxLength int) bool {
	return value != "" &&
		len(value) <= maxLength &&
		idempotencyKeyPattern.MatchString(value)
}

func defaultReason(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func actorPointer(actorUserID int64) *int64 {
	actor := actorUserID
	return &actor
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func isWorkState(value string) bool {
	switch value {
	case task.WorkStateOpen,
		task.WorkStateInProgress,
		task.WorkStatePendingVerification,
		task.WorkStateCompleted:
		return true
	default:
		return false
	}
}

func validateTask(t *task.Task) error {
	if t == nil {
		return fmt.Errorf("%w: request is required", ErrInvalidTask)
	}
	t.Title = strings.TrimSpace(t.Title)
	t.Description = strings.TrimSpace(t.Description)
	t.Category = strings.TrimSpace(t.Category)
	t.Region = strings.TrimSpace(t.Region)
	t.LocationDetail = strings.TrimSpace(t.LocationDetail)
	t.VolunteerMode = strings.TrimSpace(t.VolunteerMode)

	switch {
	case t.CreatorID <= 0:
		return fmt.Errorf("%w: authenticated creator is required", ErrInvalidTask)
	case t.Title == "" || len(t.Title) > 255:
		return fmt.Errorf("%w: title is required and must not exceed 255 characters", ErrInvalidTask)
	case t.Description == "":
		return fmt.Errorf("%w: description is required", ErrInvalidTask)
	case t.Category == "" || len(t.Category) > 100:
		return fmt.Errorf("%w: category is required and must not exceed 100 characters", ErrInvalidTask)
	case t.Region == "" || len(t.Region) > 100:
		return fmt.Errorf("%w: region is required and must not exceed 100 characters", ErrInvalidTask)
	case len(t.LocationDetail) > 255:
		return fmt.Errorf("%w: location_detail must not exceed 255 characters", ErrInvalidTask)
	case t.GoalSats < 0:
		return fmt.Errorf("%w: goal_sats must not be negative", ErrInvalidTask)
	case t.MaxVolunteers < 0:
		return fmt.Errorf("%w: max_volunteers must not be negative", ErrInvalidTask)
	case t.VolunteerMode != "open" && t.VolunteerMode != "approval_required":
		return fmt.Errorf("%w: volunteer_mode must be open or approval_required", ErrInvalidTask)
	}
	return nil
}

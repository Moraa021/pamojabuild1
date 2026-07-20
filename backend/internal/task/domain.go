package task

import (
	"context"
	"errors"
	"time"
)

var (
	ErrSlugTaken           = errors.New("task slug already exists")
	ErrStateConflict       = errors.New("task state changed or transition is invalid")
	ErrIdempotencyConflict = errors.New("idempotency key was already used for another transition")
)

const (
	WorkStateOpen                = "open"
	WorkStateInProgress          = "in_progress"
	WorkStatePendingVerification = "pending_verification"
	WorkStateCompleted           = "completed"

	FinancialStateActive           = "ACTIVE"
	FinancialStateLiquidating      = "LIQUIDATING"
	FinancialStateReadyForPayout   = "READY_FOR_PAYOUT"
	FinancialStatePayoutProcessing = "PAYOUT_PROCESSING"
	FinancialStateArchived         = "ARCHIVED"
	FinancialStateSystemLockdown   = "SYSTEM_LOCKDOWN"

	StateKindWork      = "work"
	StateKindFinancial = "financial"
)

type Task struct {
	ID                    int64
	Slug                  string
	CreatorID             int64
	Title                 string
	Description           string
	Category              string
	Region                string
	LocationDetail        string
	Status                string
	FinancialState        string
	WorkStateVersion      int64
	FinancialStateVersion int64
	GoalSats              int64
	MaxVolunteers         int64
	VolunteerMode         string
	ImagePath             string
	CreatedAt             time.Time
}

type StateTransitionCommand struct {
	TaskSlug       string
	StateKind      string
	ExpectedState  string
	TargetState    string
	ActorUserID    *int64
	Reason         string
	IdempotencyKey string
}

type StateTransition struct {
	ID             int64
	TaskSlug       string
	StateKind      string
	FromState      *string
	ToState        string
	Version        int64
	ActorUserID    *int64
	Reason         string
	IdempotencyKey string
	CreatedAt      time.Time
}

type StateTransitionResult struct {
	Transitions []StateTransition
	Replayed    bool
}

type StateHistoryOptions struct {
	StateKind string
	Page      int
	PageSize  int
}

type StateHistoryResult struct {
	Transitions []StateTransition
	Total       int
}

// RelationshipReader exposes task-specific readiness and conflict facts to the
// task state machine without allowing the task repository to own volunteer or
// trustee persistence.
type RelationshipReader interface {
	HasApprovedVolunteer(ctx context.Context, taskSlug string) (bool, error)
	ApprovedSubmissionCounts(ctx context.Context, taskSlug string) (approved int, submitted int, err error)
	IsTaskTrustee(ctx context.Context, taskSlug string, userID int64) (bool, error)
	IsTaskVolunteer(ctx context.Context, taskSlug string, userID int64) (bool, error)
}

type ListOptions struct {
	Category string
	Region   string
	Status   string
	Page     int
	PageSize int
}

type ListResult struct {
	Tasks []Task
	Total int
}

type Repository interface {
	Create(ctx context.Context, t *Task) error
	GetByID(ctx context.Context, id int64) (*Task, error)
	GetBySlug(ctx context.Context, slug string) (*Task, error)
	List(ctx context.Context, options ListOptions) (*ListResult, error)
	ApplyStateTransitions(ctx context.Context, commands []StateTransitionCommand) (*StateTransitionResult, error)
	ListStateTransitions(ctx context.Context, taskSlug string, options StateHistoryOptions) (*StateHistoryResult, error)
}

type Service interface {
	CreateCampaign(ctx context.Context, req *Task) (*Task, error)
	GetTask(ctx context.Context, slug string) (*Task, error)
	ListTasks(ctx context.Context, options ListOptions) (*ListResult, error)
	StartTask(ctx context.Context, slug string, actorUserID int64, reason, idempotencyKey string) (*StateTransitionResult, error)
	SubmitForVerification(ctx context.Context, slug string, actorUserID int64, reason, idempotencyKey string) (*StateTransitionResult, error)
	VerifyTask(ctx context.Context, slug string, actorUserID int64, reason, idempotencyKey string) (*StateTransitionResult, error)
	AdvanceFinancialState(ctx context.Context, slug, targetState, reason, idempotencyKey string) (*StateTransitionResult, error)
	ListStateTransitions(ctx context.Context, slug string, options StateHistoryOptions) (*StateHistoryResult, error)
}

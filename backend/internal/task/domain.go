package task

import (
	"context"
	"time"
)

// Task represents the core domain model.
// Because we don't have a global models.go, the model lives right here.
type Task struct {
	ID             int64
	Slug           string
	CreatorID      int64
	Title          string
	Description    string
	Category       string
	Region         string
	LocationDetail string
	Status         string // "open", "in_progress", etc.
	FinancialState string // "ACTIVE", "LIQUIDATING", etc.
	GoalSats       int64
	MaxVolunteers  int64
	VolunteerMode  string
	ImagePath      string
	CreatedAt      time.Time
}

// Repository defines the contract for database operations.
// The repository folder must implement these exact methods.
type Repository interface {
	Create(ctx context.Context, t *Task) error
	GetByID(ctx context.Context, id int64) (*Task, error)
	GetBySlug(ctx context.Context, slug string) (*Task, error)
	UpdateStatus(ctx context.Context, slug string, status string) error
	UpdateFinancialState(ctx context.Context, slug string, state string) error
}

// Service defines the contract for business logic and state guardrails.
// The service folder must implement these exact methods.
type Service interface {
	CreateCampaign(ctx context.Context, req *Task) (*Task, error)
	TransitionVolunteerStatus(ctx context.Context, slug string, targetStatus string) error
	TransitionFinancialState(ctx context.Context, slug string, targetState string) error
}

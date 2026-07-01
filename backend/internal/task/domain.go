package task

import (
	"context"
	"time"
)

type Task struct {
	ID             int64
	Slug           string
	CreatorID      int64
	Title          string
	Description    string
	Category       string
	Region         string
	LocationDetail string
	Status         string
	FinancialState string
	GoalSats       int64
	MaxVolunteers  int64
	VolunteerMode  string
	ImagePath      string
	CreatedAt      time.Time
}

type Repository interface {
	Create(ctx context.Context, t *Task) error
	GetByID(ctx context.Context, id int64) (*Task, error)
	GetBySlug(ctx context.Context, slug string) (*Task, error)
	List(ctx context.Context, category, region, status string) ([]Task, error)
	UpdateStatus(ctx context.Context, slug string, status string) error
	UpdateFinancialState(ctx context.Context, slug string, state string) error
}

type Service interface {
	CreateCampaign(ctx context.Context, req *Task) (*Task, error)
	GetTask(ctx context.Context, slug string) (*Task, error)
	ListTasks(ctx context.Context, category, region, status string) ([]Task, error)
	TransitionVolunteerStatus(ctx context.Context, slug string, targetStatus string) error
	TransitionFinancialState(ctx context.Context, slug string, targetState string) error
}

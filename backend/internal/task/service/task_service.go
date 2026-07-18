package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gosimple/slug"

	"pamojabuild1/backend/internal/events"
	"pamojabuild1/backend/internal/task"
)

var (
	ErrTaskNotFound      = errors.New("task not found")
	ErrTaskConflict      = errors.New("a task with this generated slug already exists")
	ErrInvalidTask       = errors.New("invalid task")
	ErrInvalidTaskFilter = errors.New("invalid task list filters")
)

type TaskService struct {
	repo     task.Repository
	eventBus *events.EventBus
}

func NewTaskService(repo task.Repository, eventBus *events.EventBus) *TaskService {
	return &TaskService{repo: repo, eventBus: eventBus}
}

func (s *TaskService) CreateCampaign(ctx context.Context, t *task.Task) (*task.Task, error) {
	if err := validateTask(t); err != nil {
		return nil, err
	}
	t.Slug = slug.Make(t.Title)
	if t.Slug == "" {
		return nil, fmt.Errorf("%w: title must contain letters or numbers", ErrInvalidTask)
	}
	t.Status = "open"
	t.FinancialState = "ACTIVE"
	t.CreatedAt = time.Now().UTC()

	if err := s.repo.Create(ctx, t); err != nil {
		if errors.Is(err, task.ErrSlugTaken) {
			return nil, ErrTaskConflict
		}
		return nil, err
	}

	// Publish TaskCreated event for Phase 4 event-driven flows
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

func (s *TaskService) GetTask(ctx context.Context, slug string) (*task.Task, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" || len(slug) > 255 {
		return nil, ErrTaskNotFound
	}
	t, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("get task: %w", err)
	}
	return t, nil
}

func (s *TaskService) ListTasks(ctx context.Context, options task.ListOptions) (*task.ListResult, error) {
	options.Category = strings.TrimSpace(options.Category)
	options.Region = strings.TrimSpace(options.Region)
	options.Status = strings.TrimSpace(options.Status)
	if options.Page < 1 || options.PageSize < 1 || options.PageSize > 100 ||
		len(options.Category) > 100 || len(options.Region) > 100 || len(options.Status) > 50 {
		return nil, ErrInvalidTaskFilter
	}
	return s.repo.List(ctx, options)
}

func (s *TaskService) TransitionVolunteerStatus(ctx context.Context, slug string, targetStatus string) error {
	if err := s.repo.UpdateStatus(ctx, slug, targetStatus); err != nil {
		return err
	}

	s.eventBus.Publish(events.Event{
		Type: events.TaskStatusChanged,
		Payload: events.TaskStatusChangedPayload{
			TaskSlug:  slug,
			NewStatus: targetStatus,
		},
	})

	return nil
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

func (s *TaskService) TransitionFinancialState(ctx context.Context, slug string, targetState string) error {
	task, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return err
	}

	oldState := task.FinancialState

	if err := s.repo.UpdateFinancialState(ctx, slug, targetState); err != nil {
		return err
	}

	s.eventBus.Publish(events.Event{
		Type: events.FinancialStateChanged,
		Payload: events.FinancialStateChangedPayload{
			TaskSlug: slug,
			OldState: oldState,
			NewState: targetState,
		},
	})

	return nil
}

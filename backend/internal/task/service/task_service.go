package service

import (
	"context"
	"errors"

	"github.com/gosimple/slug"

	"pamojabuild1/backend/internal/events"
	"pamojabuild1/backend/internal/task"
)

var (
	ErrTaskNotFound = errors.New("task not found")
)

type TaskService struct {
	repo     task.Repository
	eventBus *events.EventBus
}

func NewTaskService(repo task.Repository, eventBus *events.EventBus) *TaskService {
	return &TaskService{repo: repo, eventBus: eventBus}
}

func (s *TaskService) CreateCampaign(ctx context.Context, t *task.Task) (*task.Task, error) {
	t.Slug = slug.Make(t.Title)
	t.Status = "open"
	t.FinancialState = "ACTIVE"

	if err := s.repo.Create(ctx, t); err != nil {
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
	t, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, ErrTaskNotFound
	}
	return t, nil
}

func (s *TaskService) ListTasks(ctx context.Context, category, region, status string) ([]task.Task, error) {
	return s.repo.List(ctx, category, region, status)
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
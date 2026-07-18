package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"pamojabuild1/backend/internal/events"
	"pamojabuild1/backend/internal/volunteer"
)

var (
	ErrAlreadyApplied          = errors.New("already applied for this task")
	ErrInvalidApplication      = errors.New("invalid task application")
	ErrApplicationTaskNotFound = errors.New("application task not found")
	ErrRelationshipConflict    = errors.New("account cannot be both trustee and volunteer for the same task")
)

type ApplicationService struct {
	applicationRepo volunteer.ApplicationRepository
	eventBus        *events.EventBus
}

func NewApplicationService(applicationRepo volunteer.ApplicationRepository, eventBus *events.EventBus) *ApplicationService {
	return &ApplicationService{applicationRepo: applicationRepo, eventBus: eventBus}
}

func (s *ApplicationService) ApplyForTask(ctx context.Context, taskSlug string, volunteerID int64, message string) (*volunteer.TaskApplication, error) {
	taskSlug = strings.TrimSpace(taskSlug)
	message = strings.TrimSpace(message)
	if taskSlug == "" || len(taskSlug) > 255 || volunteerID <= 0 || message == "" || len(message) > 2000 {
		return nil, ErrInvalidApplication
	}

	existing, err := s.applicationRepo.GetApplicationByTaskSlug(ctx, taskSlug, volunteerID)
	if existing != nil {
		return nil, ErrAlreadyApplied
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("check existing application: %w", err)
	}

	app := &volunteer.TaskApplication{
		TaskSlug:    taskSlug,
		VolunteerID: volunteerID,
		Message:     message,
	}

	if err := s.applicationRepo.CreateApplication(ctx, app); err != nil {
		switch {
		case errors.Is(err, volunteer.ErrApplicationAlreadyExists):
			return nil, ErrAlreadyApplied
		case errors.Is(err, volunteer.ErrTaskDoesNotExist):
			return nil, ErrApplicationTaskNotFound
		case errors.Is(err, volunteer.ErrTaskRelationshipConflict):
			return nil, ErrRelationshipConflict
		default:
			return nil, fmt.Errorf("create application: %w", err)
		}
	}

	if s.eventBus != nil {
		s.eventBus.Publish(events.Event{
			Type: events.ApplicationSubmitted,
			Payload: events.ApplicationSubmittedPayload{
				TaskSlug:    taskSlug,
				VolunteerID: volunteerID,
			},
		})
	}

	return app, nil
}

func (s *ApplicationService) GetApplications(ctx context.Context, volunteerID int64) ([]volunteer.TaskApplication, error) {
	return s.applicationRepo.GetApplicationsByVolunteerID(ctx, volunteerID)
}

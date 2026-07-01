package service

import (
	"context"
	"errors"

	"pamojabuild1/backend/internal/events"
	"pamojabuild1/backend/internal/volunteer"
)

var (
	ErrAlreadyApplied = errors.New("already applied for this task")
)

type ApplicationService struct {
	applicationRepo volunteer.ApplicationRepository
	eventBus        *events.EventBus
}

func NewApplicationService(applicationRepo volunteer.ApplicationRepository, eventBus *events.EventBus) *ApplicationService {
	return &ApplicationService{applicationRepo: applicationRepo, eventBus: eventBus}
}

func (s *ApplicationService) ApplyForTask(ctx context.Context, taskSlug string, volunteerID int64, message string) (*volunteer.TaskApplication, error) {
	existing, _ := s.applicationRepo.GetApplicationByTaskSlug(ctx, taskSlug, volunteerID)
	if existing != nil {
		return nil, ErrAlreadyApplied
	}

	app := &volunteer.TaskApplication{
		TaskSlug:    taskSlug,
		VolunteerID: volunteerID,
		Message:     message,
	}

	if err := s.applicationRepo.CreateApplication(ctx, app); err != nil {
		return nil, err
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

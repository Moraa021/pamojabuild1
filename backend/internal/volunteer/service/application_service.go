package service

import (
	"context"
	"errors"

	"pamojabuild1/backend/internal/volunteer"
)

var (
	ErrAlreadyApplied = errors.New("already applied for this task")
)

type ApplicationService struct {
	applicationRepo volunteer.ApplicationRepository
}

func NewApplicationService(applicationRepo volunteer.ApplicationRepository) *ApplicationService {
	return &ApplicationService{applicationRepo: applicationRepo}
}

func (s *ApplicationService) ApplyForTask(ctx context.Context, taskSlug string, volunteerID int64, message string) (*volunteer.TaskApplication, error) {
	existing, _ := s.applicationRepo.GetByTaskSlug(ctx, taskSlug, volunteerID)
	if existing != nil {
		return nil, ErrAlreadyApplied
	}

	app := &volunteer.TaskApplication{
		TaskSlug:    taskSlug,
		VolunteerID: volunteerID,
		Message:     message,
	}

	if err := s.applicationRepo.Create(ctx, app); err != nil {
		return nil, err
	}

	return app, nil
}

func (s *ApplicationService) GetApplications(ctx context.Context, volunteerID int64) ([]volunteer.TaskApplication, error) {
	return s.applicationRepo.GetByVolunteerID(ctx, volunteerID)
}

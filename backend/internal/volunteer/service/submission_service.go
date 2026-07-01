package service

import (
	"context"
	"errors"

	"pamojabuild1/backend/internal/events"
	"pamojabuild1/backend/internal/volunteer"
)

var (
	ErrNotApproved = errors.New("application not approved for this task")
)

type SubmissionService struct {
	submissionRepo  volunteer.SubmissionRepository
	applicationRepo volunteer.ApplicationRepository
	eventBus        *events.EventBus
}

func NewSubmissionService(
	submissionRepo volunteer.SubmissionRepository,
	applicationRepo volunteer.ApplicationRepository,
	eventBus *events.EventBus,
) *SubmissionService {
	return &SubmissionService{
		submissionRepo:  submissionRepo,
		applicationRepo: applicationRepo,
		eventBus:        eventBus,
	}
}

func (s *SubmissionService) SubmitWork(ctx context.Context, taskSlug string, volunteerID int64, description string, evidenceURLs []string) (*volunteer.TaskSubmission, error) {
	app, err := s.applicationRepo.GetApplicationByTaskSlug(ctx, taskSlug, volunteerID)
	if err != nil || app.Status != "approved" {
		return nil, ErrNotApproved
	}

	sub := &volunteer.TaskSubmission{
		TaskSlug:     taskSlug,
		VolunteerID:  volunteerID,
		Description:  description,
		EvidenceURLs: evidenceURLs,
	}

	if err := s.submissionRepo.CreateSubmission(ctx, sub); err != nil {
		return nil, err
	}

	if s.eventBus != nil {
		s.eventBus.Publish(events.Event{
			Type: events.SubmissionCreated,
			Payload: events.SubmissionCreatedPayload{
				TaskSlug:    taskSlug,
				VolunteerID: volunteerID,
				Description: description,
			},
		})
	}

	return sub, nil
}

func (s *SubmissionService) GetSubmissions(ctx context.Context, volunteerID int64) ([]volunteer.TaskSubmission, error) {
	return s.submissionRepo.GetSubmissionsByVolunteerID(ctx, volunteerID)
}

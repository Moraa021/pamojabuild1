package service

import (
	"context"
	"errors"

	"pamojabuild1/backend/internal/volunteer"
)

var (
	ErrNotApproved = errors.New("application not approved for this task")
)

type SubmissionService struct {
	submissionRepo  volunteer.SubmissionRepository
	applicationRepo volunteer.ApplicationRepository
}

func NewSubmissionService(
	submissionRepo volunteer.SubmissionRepository,
	applicationRepo volunteer.ApplicationRepository,
) *SubmissionService {
	return &SubmissionService{
		submissionRepo:  submissionRepo,
		applicationRepo: applicationRepo,
	}
}

func (s *SubmissionService) SubmitWork(ctx context.Context, taskSlug string, volunteerID int64, description string, evidenceURLs []string) (*volunteer.TaskSubmission, error) {
	app, err := s.applicationRepo.GetByTaskSlug(ctx, taskSlug, volunteerID)
	if err != nil || app.Status != "approved" {
		return nil, ErrNotApproved
	}

	sub := &volunteer.TaskSubmission{
		TaskSlug:     taskSlug,
		VolunteerID:  volunteerID,
		Description:  description,
		EvidenceURLs: evidenceURLs,
	}

	if err := s.submissionRepo.Create(ctx, sub); err != nil {
		return nil, err
	}

	return sub, nil
}

func (s *SubmissionService) GetSubmissions(ctx context.Context, volunteerID int64) ([]volunteer.TaskSubmission, error) {
	return s.submissionRepo.GetByVolunteerID(ctx, volunteerID)
}

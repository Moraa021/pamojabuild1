package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"pamojabuild1/backend/internal/events"
	"pamojabuild1/backend/internal/volunteer"
)

var (
	ErrNotApproved       = errors.New("application not approved for this task")
	ErrInvalidSubmission = errors.New("invalid task submission")
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
	taskSlug = strings.TrimSpace(taskSlug)
	description = strings.TrimSpace(description)
	if taskSlug == "" || len(taskSlug) > 255 || volunteerID <= 0 || description == "" || len(description) > 5000 {
		return nil, ErrInvalidSubmission
	}
	if len(evidenceURLs) == 0 || len(evidenceURLs) > 20 {
		return nil, ErrInvalidSubmission
	}
	for index := range evidenceURLs {
		evidenceURLs[index] = strings.TrimSpace(evidenceURLs[index])
		parsed, err := url.ParseRequestURI(evidenceURLs[index])
		if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" || len(evidenceURLs[index]) > 2048 {
			return nil, fmt.Errorf("%w: evidence_urls must contain valid HTTP(S) URLs", ErrInvalidSubmission)
		}
	}

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

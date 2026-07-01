package service

import (
	"context"
	"errors"

	"pamojabuild1/backend/internal/volunteer"
)

var (
	ErrProfileNotFound = errors.New("volunteer profile not found")
	ErrAlreadyApplied  = errors.New("already applied for this task")
	ErrNotApproved     = errors.New("application not approved")
)

type VolunteerService struct {
	profileRepo     volunteer.ProfileRepository
	applicationRepo volunteer.ApplicationRepository
	submissionRepo  volunteer.SubmissionRepository
	paymentRepo     volunteer.PaymentRepository
}

func NewVolunteerService(
	profileRepo volunteer.ProfileRepository,
	applicationRepo volunteer.ApplicationRepository,
	submissionRepo volunteer.SubmissionRepository,
	paymentRepo volunteer.PaymentRepository,
) *VolunteerService {
	return &VolunteerService{
		profileRepo:     profileRepo,
		applicationRepo: applicationRepo,
		submissionRepo:  submissionRepo,
		paymentRepo:     paymentRepo,
	}
}

func (s *VolunteerService) GetProfile(ctx context.Context, userID int64) (*volunteer.VolunteerProfile, error) {
	profile, err := s.profileRepo.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, ErrProfileNotFound
	}
	return profile, nil
}

func (s *VolunteerService) UpdateProfile(ctx context.Context, userID int64, req *volunteer.VolunteerProfile) error {
	req.UserID = userID
	return s.profileRepo.UpdateProfile(ctx, req)
}

func (s *VolunteerService) ApplyForTask(ctx context.Context, taskSlug string, volunteerID int64, message string) (*volunteer.TaskApplication, error) {
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

func (s *VolunteerService) SubmitWork(ctx context.Context, taskSlug string, volunteerID int64, description string, evidenceURLs []string) (*volunteer.TaskSubmission, error) {
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

func (s *VolunteerService) GetPayments(ctx context.Context, volunteerID int64) ([]volunteer.Payment, error) {
	return s.paymentRepo.GetPaymentsByVolunteerID(ctx, volunteerID)
}
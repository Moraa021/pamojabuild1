package service

import (
	"context"

	"pamojabuild1/backend/internal/volunteer"
)

type ReputationService struct {
	profileRepo     volunteer.ProfileRepository
	applicationRepo volunteer.ApplicationRepository
	submissionRepo  volunteer.SubmissionRepository
	paymentRepo     volunteer.PaymentRepository
}

func NewReputationService(
	profileRepo volunteer.ProfileRepository,
	applicationRepo volunteer.ApplicationRepository,
	submissionRepo volunteer.SubmissionRepository,
	paymentRepo volunteer.PaymentRepository,
) *ReputationService {
	return &ReputationService{
		profileRepo:     profileRepo,
		applicationRepo: applicationRepo,
		submissionRepo:  submissionRepo,
		paymentRepo:     paymentRepo,
	}
}

func (s *ReputationService) CalculateReputation(ctx context.Context, userID int64) (*volunteer.ReputationResponse, error) {
	profile, err := s.profileRepo.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	applications, _ := s.applicationRepo.GetByVolunteerID(ctx, userID)
	submissions, _ := s.submissionRepo.GetByVolunteerID(ctx, userID)
	payments, _ := s.paymentRepo.GetPaymentsByVolunteerID(ctx, userID)

	// Calculate success rate
	completedSubmissions := 0
	for _, sub := range submissions {
		if sub.Status == "approved" || sub.Status == "verified" {
			completedSubmissions++
		}
	}

	successRate := 0.0
	if len(applications) > 0 {
		successRate = float64(completedSubmissions) / float64(len(applications)) * 100
	}

	// Calculate tier based on score
	tier := calculateTier(profile.ReputationScore)

	return &volunteer.ReputationResponse{
		UserID:          userID,
		Score:           profile.ReputationScore,
		Tier:            tier,
		CompletedTasks:  profile.CompletedTasks,
		TotalEarnedSats: profile.TotalEarnedSats,
		SuccessRate:     successRate,
	}, nil
}

func calculateTier(score int) string {
	switch {
	case score >= 1000:
		return "Elite"
	case score >= 500:
		return "Expert"
	case score >= 200:
		return "Trusted"
	case score >= 50:
		return "Active"
	default:
		return "New"
	}
}
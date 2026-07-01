package service

import (
	"context"
	"errors"

	"pamojabuild1/backend/internal/volunteer"
)

var (
	ErrProfileNotFound = errors.New("volunteer profile not found")
)

type VolunteerService struct {
	profileRepo volunteer.ProfileRepository
}

func NewVolunteerService(profileRepo volunteer.ProfileRepository) *VolunteerService {
	return &VolunteerService{profileRepo: profileRepo}
}

func (s *VolunteerService) GetProfile(ctx context.Context, userID int64) (*volunteer.VolunteerProfile, error) {
	profile, err := s.profileRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, ErrProfileNotFound
	}
	return profile, nil
}

func (s *VolunteerService) UpdateProfile(ctx context.Context, userID int64, req *volunteer.VolunteerProfile) error {
	req.UserID = userID
	return s.profileRepo.Update(ctx, req)
}

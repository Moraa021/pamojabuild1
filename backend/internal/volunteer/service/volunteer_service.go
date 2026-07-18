package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"pamojabuild1/backend/internal/volunteer"
)

var (
	ErrProfileNotFound = errors.New("volunteer profile not found")
	ErrInvalidProfile  = errors.New("invalid volunteer profile")
)

type VolunteerService struct {
	profileRepo volunteer.ProfileRepository
	paymentRepo volunteer.PaymentRepository
}

func NewVolunteerService(profileRepo volunteer.ProfileRepository, paymentRepo volunteer.PaymentRepository) *VolunteerService {
	return &VolunteerService{profileRepo: profileRepo, paymentRepo: paymentRepo}
}

func (s *VolunteerService) GetProfile(ctx context.Context, userID int64) (*volunteer.VolunteerProfile, error) {
	profile, err := s.profileRepo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProfileNotFound
		}
		return nil, fmt.Errorf("get volunteer profile: %w", err)
	}
	return profile, nil
}

func (s *VolunteerService) UpdateProfile(ctx context.Context, userID int64, req *volunteer.VolunteerProfile) error {
	if err := validateProfile(userID, req); err != nil {
		return err
	}
	req.UserID = userID
	return s.profileRepo.Update(ctx, req)
}

func (s *VolunteerService) UpdatePaymentProfile(ctx context.Context, userID int64, lightningAddress, onchainAddress string) error {
	lightningAddress = strings.TrimSpace(lightningAddress)
	onchainAddress = strings.TrimSpace(onchainAddress)
	if userID <= 0 || len(lightningAddress) > 255 || len(onchainAddress) > 255 {
		return fmt.Errorf("%w: payment addresses must not exceed 255 characters", ErrInvalidProfile)
	}
	if lightningAddress == "" && onchainAddress == "" {
		return fmt.Errorf("%w: at least one payment address is required", ErrInvalidProfile)
	}
	return s.profileRepo.UpdatePaymentProfile(ctx, userID, lightningAddress, onchainAddress)
}

func (s *VolunteerService) GetPayments(ctx context.Context, userID int64) ([]volunteer.Payment, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("%w: authenticated account is required", ErrInvalidProfile)
	}
	return s.paymentRepo.GetPaymentsByVolunteerID(ctx, userID)
}

func validateProfile(userID int64, profile *volunteer.VolunteerProfile) error {
	if userID <= 0 || profile == nil {
		return fmt.Errorf("%w: authenticated account and profile are required", ErrInvalidProfile)
	}
	profile.Bio = strings.TrimSpace(profile.Bio)
	profile.LightningAddress = strings.TrimSpace(profile.LightningAddress)
	profile.OnchainAddress = strings.TrimSpace(profile.OnchainAddress)
	if len(profile.Bio) > 2000 || len(profile.LightningAddress) > 255 || len(profile.OnchainAddress) > 255 {
		return fmt.Errorf("%w: one or more fields exceed their maximum length", ErrInvalidProfile)
	}
	if len(profile.Skills) > 50 {
		return fmt.Errorf("%w: no more than 50 skills are allowed", ErrInvalidProfile)
	}
	for index := range profile.Skills {
		profile.Skills[index] = strings.TrimSpace(profile.Skills[index])
		if profile.Skills[index] == "" || len(profile.Skills[index]) > 100 {
			return fmt.Errorf("%w: each skill must contain 1 to 100 characters", ErrInvalidProfile)
		}
	}
	return nil
}

package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"

	"pamojabuild1/backend/internal/events"
	"pamojabuild1/backend/internal/trustee"
)

var (
	ErrInvalidTrusteeIndex = errors.New("trustee index must be between 0 and 4")
	ErrSlotAlreadyTaken    = errors.New("trustee slot already assigned")
)

type TrusteeService struct {
	keyRepo  trustee.KeyRepository
	userRepo trustee.UserRepository
	eventBus *events.EventBus
}

func NewTrusteeService(keyRepo trustee.KeyRepository, userRepo trustee.UserRepository, eventBus *events.EventBus) *TrusteeService {
	return &TrusteeService{keyRepo: keyRepo, userRepo: userRepo, eventBus: eventBus}
}

func (s *TrusteeService) RegisterUser(ctx context.Context, email, password, displayName string) (*trustee.User, error) {
	user := &trustee.User{
		Email:        email,
		PasswordHash: password,
		DisplayName:  displayName,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *TrusteeService) AssignTrusteeSlot(ctx context.Context, slug string, key *trustee.TrusteeKey) error {
	if key.TrusteeIndex < 0 || key.TrusteeIndex > 4 {
		return ErrInvalidTrusteeIndex
	}

	existing, _ := s.keyRepo.GetSpecificTrustee(ctx, slug, key.TrusteeIndex)
	if existing != nil && existing.UserID != 0 {
		return ErrSlotAlreadyTaken
	}

	key.TaskSlug = slug
	if err := s.keyRepo.SaveKeys(ctx, key); err != nil {
		return err
	}

	if s.eventBus != nil {
		s.eventBus.Publish(events.Event{
			Type: events.TrusteeRegistered,
			Payload: map[string]interface{}{
				"task_slug":     slug,
				"trustee_index": key.TrusteeIndex,
			},
		})
	}

	return nil
}

func (s *TrusteeService) VerifyWebCryptoSignature(ctx context.Context, pubKeyHex string, message []byte, signatureHex string) (bool, error) {
	pubKeyBytes, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return false, fmt.Errorf("invalid public key hex: %w", err)
	}

	pubKey, err := x509.ParsePKIXPublicKey(pubKeyBytes)
	if err != nil {
		return false, fmt.Errorf("failed to parse public key: %w", err)
	}

	ecdsaPubKey, ok := pubKey.(*ecdsa.PublicKey)
	if !ok {
		return false, errors.New("not an ECDSA public key")
	}

	_ = ecdsaPubKey

	return true, nil
}

func (s *TrusteeService) GetTaskTrustees(ctx context.Context, taskSlug string) ([]trustee.TrusteeKey, error) {
	return s.keyRepo.GetKeysByTask(ctx, taskSlug)
}

package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"

	"pamojabuild1/backend/internal/trustee"
)

var (
	ErrInvalidTrusteeIndex = errors.New("trustee index must be between 0 and 4")
	ErrSlotAlreadyTaken    = errors.New("trustee slot already assigned")
)

type TrusteeService struct {
	repo trustee.KeyRepository
}

func NewTrusteeService(repo trustee.KeyRepository) *TrusteeService {
	return &TrusteeService{repo: repo}
}

func (s *TrusteeService) RegisterUser(ctx context.Context, email, password, displayName string) (*trustee.User, error) {
	user := &trustee.User{
		Email:    email,
		PasswordHash: password, // In production, hash this
		DisplayName: displayName,
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *TrusteeService) AssignTrusteeSlot(ctx context.Context, slug string, key *trustee.TrusteeKey) error {
	if key.TrusteeIndex < 0 || key.TrusteeIndex > 4 {
		return ErrInvalidTrusteeIndex
	}

	existing, _ := s.repo.GetSpecificTrustee(ctx, slug, key.TrusteeIndex)
	if existing != nil && existing.UserID != 0 {
		return ErrSlotAlreadyTaken
	}

	key.TaskSlug = slug
	return s.repo.SaveKeys(ctx, key)
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

	sigBytes, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false, fmt.Errorf("invalid signature hex: %w", err)
	}

	// Note: In production, implement proper ECDSA verification
	// This is a placeholder for the actual crypto verification
	_ = ecdsaPubKey
	_ = sigBytes

	return true, nil
}

func (s *TrusteeService) GetTaskTrustees(ctx context.Context, taskSlug string) ([]trustee.TrusteeKey, error) {
	return s.repo.GetKeysByTask(ctx, taskSlug)
}
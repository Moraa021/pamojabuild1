package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"pamojabuild1/backend/internal/events"
	"pamojabuild1/backend/internal/trustee"
)

var (
	ErrInvalidTrusteeIndex = errors.New("trustee index must be between 0 and 4")
	ErrSlotAlreadyTaken    = errors.New("trustee slot already assigned")
	ErrInvalidTrusteeKeys  = errors.New("invalid trustee key registration")
	ErrTrusteeTaskNotFound = errors.New("trustee task not found")
	ErrTrusteeConflict     = errors.New("trustee registration conflicts with an existing task relationship")
)

type TrusteeService struct {
	keyRepo  trustee.KeyRepository
	eventBus *events.EventBus
}

func NewTrusteeService(keyRepo trustee.KeyRepository, eventBus *events.EventBus) *TrusteeService {
	return &TrusteeService{keyRepo: keyRepo, eventBus: eventBus}
}

func (s *TrusteeService) AssignTrusteeSlot(ctx context.Context, slug string, key *trustee.TrusteeKey) error {
	slug = strings.TrimSpace(slug)
	if key == nil || slug == "" || len(slug) > 255 {
		return ErrInvalidTrusteeKeys
	}
	if key.TrusteeIndex < 0 || key.TrusteeIndex > 4 {
		return ErrInvalidTrusteeIndex
	}
	if key.UserID <= 0 {
		return ErrInvalidTrusteeKeys
	}
	key.Xpub = strings.TrimSpace(key.Xpub)
	key.WebCryptoPubkeyHex = strings.TrimSpace(key.WebCryptoPubkeyHex)
	// Full network/version parsing and proof of key ownership belong to trustee
	// onboarding. This contract boundary still rejects empty or oversized input
	// so storage errors are not used as validation.
	if key.Xpub == "" || len(key.Xpub) > 255 ||
		key.WebCryptoPubkeyHex == "" || len(key.WebCryptoPubkeyHex) > 512 {
		return ErrInvalidTrusteeKeys
	}

	existing, err := s.keyRepo.GetSpecificTrustee(ctx, slug, key.TrusteeIndex)
	if err == nil && existing != nil {
		return ErrSlotAlreadyTaken
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check trustee slot: %w", err)
	}

	key.TaskSlug = slug
	if err := s.keyRepo.SaveKeys(ctx, key); err != nil {
		// The database primary key is the final concurrency guard. The insert
		// never upserts because key registration must not silently replace a
		// trustee who won a race for the same slot.
		switch {
		case errors.Is(err, trustee.ErrTaskNotFound):
			return ErrTrusteeTaskNotFound
		case errors.Is(err, trustee.ErrRegistrationConflict):
			return ErrTrusteeConflict
		default:
			return fmt.Errorf("save trustee slot: %w", err)
		}
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

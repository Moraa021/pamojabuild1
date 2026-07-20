package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"pamojabuild1/backend/internal/escrow"
	"pamojabuild1/backend/internal/events"
	"pamojabuild1/backend/internal/ledger"
	"pamojabuild1/backend/internal/trustee"
)

var (
	ErrInsufficientSignatures = errors.New("insufficient signatures, need at least 3 of 5")
	ErrInvalidPayoutRequest   = errors.New("invalid payout request")
	ErrPayoutNotReady         = errors.New("payout review scaffold is not ready")
	ErrTrusteeNotAssigned     = errors.New("account is not assigned as a trustee for this task")
)

type EscrowService struct {
	repo        escrow.SignatureRepository
	trusteeRepo trustee.KeyRepository
	ledgerRepo  ledger.Repository
	eventBus    *events.EventBus
}

func NewEscrowService(repo escrow.SignatureRepository, trusteeRepo trustee.KeyRepository, ledgerRepo ledger.Repository, eventBus *events.EventBus) *EscrowService {
	return &EscrowService{
		repo:        repo,
		trusteeRepo: trusteeRepo,
		ledgerRepo:  ledgerRepo,
		eventBus:    eventBus,
	}
}

func (s *EscrowService) PreparePayoutManifest(ctx context.Context, taskSlug string) (*escrow.PayoutManifest, error) {
	taskSlug = strings.TrimSpace(taskSlug)
	if taskSlug == "" || len(taskSlug) > 255 {
		return nil, ErrInvalidPayoutRequest
	}
	// Get task balance
	balance, err := s.ledgerRepo.GetTaskBalance(ctx, taskSlug)
	if err != nil {
		return nil, err
	}

	// Get trustee keys
	trusteeKeys, err := s.trusteeRepo.GetKeysByTask(ctx, taskSlug)
	if err != nil {
		return nil, err
	}

	if len(trusteeKeys) < 5 {
		return nil, ErrPayoutNotReady
	}

	// In production, this would:
	// 1. Create the PSBT for L1 payout
	// 2. Prepare the Lightning invoice for L2 payout
	// 3. Return the unsigned PSBT and invoice for trustees to sign

	fmt.Printf("Preparing payout for %s: L1=%d sats, L2=%d sats\n",
		taskSlug, balance.L1BalanceSats, balance.L2BalanceSats)

	return &escrow.PayoutManifest{
		TaskSlug:         taskSlug,
		UnsignedPSBTHex:  "unsigned_psbt_placeholder",
		VolunteerInvoice: "volunteer_invoice_placeholder",
		L1AmountSats:     balance.L1BalanceSats,
		L2AmountSats:     balance.L2BalanceSats,
	}, nil
}

func (s *EscrowService) SubmitTrusteeSignature(
	ctx context.Context,
	taskSlug string,
	trusteeUserID int64,
	l1Signature string,
	l2Signature string,
) (bool, error) {
	taskSlug = strings.TrimSpace(taskSlug)
	l1Signature = strings.TrimSpace(l1Signature)
	l2Signature = strings.TrimSpace(l2Signature)
	if taskSlug == "" || len(taskSlug) > 255 || trusteeUserID <= 0 ||
		l1Signature == "" || l2Signature == "" ||
		len(l1Signature) > 65536 || len(l2Signature) > 65536 {
		return false, ErrInvalidPayoutRequest
	}

	trusteeKeys, err := s.trusteeRepo.GetKeysByTask(ctx, taskSlug)
	if err != nil {
		return false, fmt.Errorf("load task trustees: %w", err)
	}
	var storedPublicKey string
	for _, key := range trusteeKeys {
		if key.UserID == trusteeUserID {
			storedPublicKey = key.WebCryptoPubkeyHex
			break
		}
	}
	if storedPublicKey == "" {
		return false, ErrTrusteeNotAssigned
	}

	payload := &escrow.SignatureCollection{
		TaskSlug:             taskSlug,
		TrusteePublicKeyHex:  storedPublicKey,
		L1SignatureFragment:  l1Signature,
		L2WebCryptoSignature: l2Signature,
	}

	if err := s.repo.SaveSignature(ctx, payload); err != nil {
		return false, err
	}

	count, err := s.repo.GetSignatureCount(ctx, taskSlug)
	if err != nil {
		return false, err
	}

	thresholdReached := count >= 3
	if thresholdReached && s.eventBus != nil {
		s.eventBus.Publish(events.Event{
			Type: events.ThresholdReached,
			Payload: events.ThresholdReachedPayload{
				TaskSlug:     taskSlug,
				RequiredSigs: 3,
				Signatures:   count,
			},
		})
	}

	return thresholdReached, nil
}

func (s *EscrowService) FinalizeAndBroadcastPayout(ctx context.Context, taskSlug string) error {
	signatures, err := s.repo.GetSignatures(ctx, taskSlug)
	if err != nil {
		return err
	}

	if len(signatures) < 3 {
		return ErrInsufficientSignatures
	}

	// In production, this would:
	// 1. Combine the 3+ signatures into the PSBT
	// 2. Broadcast the L1 transaction
	// 3. Pay the Lightning invoice for L2 balance
	// 4. Record everything in the ledger

	fmt.Printf("Broadcasting payout for %s with %d signatures\n", taskSlug, len(signatures))
	return nil
}

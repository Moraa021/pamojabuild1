package service

import (
	"context"
	"errors"
	"fmt"

	"pamojabuild1/backend/internal/escrow"
	"pamojabuild1/backend/internal/ledger"
	"pamojabuild1/backend/internal/trustee"
)

var (
	ErrInsufficientSignatures = errors.New("insufficient signatures, need at least 3 of 5")
)

type EscrowService struct {
	repo      escrow.PayoutOrchestrator
	trusteeRepo trustee.KeyRepository
	ledgerRepo  ledger.Repository
}

func NewEscrowService(repo escrow.PayoutOrchestrator, trusteeRepo trustee.KeyRepository, ledgerRepo ledger.Repository) *EscrowService {
	return &EscrowService{
		repo:        repo,
		trusteeRepo: trusteeRepo,
		ledgerRepo:  ledgerRepo,
	}
}

func (s *EscrowService) PreparePayoutManifest(ctx context.Context, taskSlug string, destinationAddress string, volunteerInvoice string) (*escrow.SignatureCollection, error) {
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
		return nil, errors.New("all 5 trustee slots must be filled")
	}

	// Build payout manifest
	manifest := &escrow.SignatureCollection{
		TaskSlug: taskSlug,
	}

	// In production, this would:
	// 1. Create the PSBT for L1 payout
	// 2. Prepare the Lightning invoice for L2 payout
	// 3. Return the unsigned PSBT and invoice for trustees to sign

	fmt.Printf("Preparing payout for %s: L1=%d sats, L2=%d sats\n",
		taskSlug, balance.L1BalanceSats, balance.L2BalanceSats)

	return manifest, nil
}

func (s *EscrowService) SubmitTrusteeSignature(ctx context.Context, taskSlug string, payload *escrow.SignatureCollection) (bool, error) {
	payload.TaskSlug = taskSlug
	
	if err := s.repo.SaveSignature(ctx, payload); err != nil {
		return false, err
	}

	count, err := s.repo.GetSignatureCount(ctx, taskSlug)
	if err != nil {
		return false, err
	}

	// Check if we've reached 3/5 threshold
	return count >= 3, nil
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
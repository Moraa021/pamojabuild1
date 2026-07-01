package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"

	"pamojabuild1/backend/internal/events"
	"pamojabuild1/backend/internal/ledger"
)

var (
	ErrChainIntegrity = errors.New("ledger chain integrity violation detected")
)

type LedgerService struct {
	repo         ledger.Repository
	serverSecret string
	mu           sync.Mutex
	eventBus     *events.EventBus
}

func NewLedgerService(repo ledger.Repository, serverSecret string, eventBus *events.EventBus) *LedgerService {
	return &LedgerService{repo: repo, serverSecret: serverSecret, eventBus: eventBus}
}

func (s *LedgerService) CalculateRowHMAC(entry *ledger.LedgerEntry, previousHash []byte, serverSecret string) ([]byte, error) {
	mac := hmac.New(sha256.New, []byte(serverSecret))
	
	data := fmt.Sprintf("%s:%s:%d:%s", entry.TaskSlug, entry.EntryType, entry.AmountSats, entry.ReferenceID)
	mac.Write([]byte(data))
	
	if previousHash != nil {
		mac.Write(previousHash)
	}

	return mac.Sum(nil), nil
}

func (s *LedgerService) VerifyEntireChainIntegrity(ctx context.Context, taskSlug string) (bool, error) {
    entries, err := s.repo.GetAllEntries(ctx, taskSlug)
    if err != nil {
        return false, err
    }

    if len(entries) == 0 {
        return true, nil
    }

    var previousHash []byte
    for _, entry := range entries {
        expectedHMAC, err := s.CalculateRowHMAC(&entry, previousHash, s.serverSecret)
		if err != nil {
			return false, err
		}

		if !hmac.Equal(expectedHMAC, entry.RowHMAC) {
			return false, ErrChainIntegrity
		}

		previousHash = entry.RowHMAC
	}

	return true, nil
}

func (s *LedgerService) RecordValidatedTransaction(ctx context.Context, taskSlug string, entryType string, amountSats int64, refID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	lastEntry, err := s.repo.GetLastEntry(ctx, taskSlug)
	if err != nil && err.Error() != "sql: no rows in result set" {
		return err
	}

	var previousHash []byte
	if lastEntry != nil {
		previousHash = lastEntry.RowHMAC
	}

	entry := &ledger.LedgerEntry{
		TaskSlug:     taskSlug,
		EntryType:    entryType,
		AmountSats:   amountSats,
		ReferenceID:  refID,
		PreviousHash: previousHash,
	}

	rowHMAC, err := s.CalculateRowHMAC(entry, previousHash, s.serverSecret)
	if err != nil {
		return err
	}

	entry.RowHMAC = rowHMAC
	if err := s.repo.AppendEntry(ctx, entry); err != nil {
		return err
	}

	if s.eventBus != nil {
		s.eventBus.Publish(events.Event{
			Type: events.TransactionRecorded,
			Payload: map[string]interface{}{
				"task_slug":  taskSlug,
				"entry_type": entryType,
				"amount_sats": amountSats,
				"reference_id": refID,
			},
		})
	}

	return nil
}

func (s *LedgerService) GetTaskBalance(ctx context.Context, taskSlug string) (*ledger.BalanceSummary, error) {
	return s.repo.GetTaskBalance(ctx, taskSlug)
}

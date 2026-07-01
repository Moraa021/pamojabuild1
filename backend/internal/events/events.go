package events

import "time"

type EventType string

const (
	PaymentSettled      EventType = "payment.settled"
	ThresholdReached    EventType = "escrow.threshold_reached"
	PayoutFinalized     EventType = "escrow.payout_finalized"
	VolunteerVerified   EventType = "volunteer.verified"
	TaskStatusChanged   EventType = "task.status_changed"
	FinancialStateChanged EventType = "task.financial_state_changed"
)

type Event struct {
	Type      EventType
	Payload   interface{}
	Timestamp time.Time
}

type PaymentSettledPayload struct {
	TaskSlug    string
	AmountSats  int64
	PaymentHash string
}

type ThresholdReachedPayload struct {
	TaskSlug       string
	Signatures     int
	RequiredSigs   int
}

type PayoutFinalizedPayload struct {
	TaskSlug      string
	L1TxID        string
	L2PaymentHash string
	TotalPaidSats int64
}

type TaskStatusChangedPayload struct {
	TaskSlug   string
	OldStatus  string
	NewStatus  string
}

type FinancialStateChangedPayload struct {
	TaskSlug    string
	OldState    string
	NewState    string
}
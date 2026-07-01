package events

import "time"

type EventType string

const (
    // Phase 4 event topics
    TaskCreated          EventType = "task.created"
    DonationReceived     EventType = "donation.received"
    ThresholdReached     EventType = "threshold.reached"
    TaskLiquidating      EventType = "task.liquidating"
    TaskReadyForPayout   EventType = "task.ready_for_payout"
    TrusteeSigned        EventType = "trustee.signed"
    TrusteeRegistered    EventType = "trustee.registered"
    PayoutCompleted      EventType = "payout.completed"
    TaskArchived         EventType = "task.archived"
    VolunteerVerified    EventType = "volunteer.verified"
    ApplicationSubmitted EventType = "application.submitted"
    SubmissionCreated    EventType = "submission.created"

    // Internal operational events
    PaymentSettled        EventType = "payment.settled"
    TransactionRecorded   EventType = "transaction.recorded"
    FinancialStateChanged EventType = "financial.state_changed"
    TaskStatusChanged     EventType = "task.status.changed"
)

type Event struct {
    Type      EventType
    Payload   interface{}
    Timestamp time.Time
}

// Payload types for event bus subscribers.

type TaskCreatedPayload struct {
    TaskSlug      string
    CreatorUserID int64
    Title         string
    Category      string
    Region        string
    GoalSats      int64
}

type DonationReceivedPayload struct {
    TaskSlug   string
    AmountSats int64
    DonorID    int64
    Source     string
}

type ThresholdReachedPayload struct {
    TaskSlug     string
    RequiredSigs int
    Signatures   int
}

type TaskLiquidatingPayload struct {
    TaskSlug string
    Reason   string
}

type TaskReadyForPayoutPayload struct {
    TaskSlug    string
    AmountSats  int64
    PayoutData  string
}

type TrusteeSignedPayload struct {
    TaskSlug            string
    TrusteePublicKeyHex string
    SignatureFragment   string
}

type PayoutCompletedPayload struct {
    TaskSlug      string
    L1TxID        string
    L2PaymentHash string
    TotalPaidSats int64
}

type TaskArchivedPayload struct {
    TaskSlug string
    Reason   string
}

type ApplicationSubmittedPayload struct {
    TaskSlug    string
    VolunteerID int64
}

type SubmissionCreatedPayload struct {
    TaskSlug    string
    VolunteerID int64
    Description string
}

type PaymentSettledPayload struct {
    TaskSlug    string
    AmountSats  int64
    PaymentHash string
}

type FinancialStateChangedPayload struct {
    TaskSlug string
    OldState string
    NewState string
}

type TaskStatusChangedPayload struct {
    TaskSlug  string
    NewStatus string
}

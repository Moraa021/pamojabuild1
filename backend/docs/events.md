# Event Bus and Payload Documentation

This document describes the internal event topics and payload structures used by the backend event bus.

## Overview

The backend uses an in-memory event bus in `internal/events/bus.go` to decouple domain services. Events are published using an `Event` envelope and delivered synchronously to subscribers.

The event bus also stores history and supports replay via `Replay()`, which replays stored events in order to current subscribers.

## Event Envelope

```go
package events

type EventType string

type Event struct {
    Type      EventType
    Payload   interface{}
    Timestamp time.Time
}
```

- `Type`: event topic string
- `Payload`: data structure associated with the event
- `Timestamp`: event publish time

## Event Topics and Payloads

### TaskCreated

- Topic: `task.created`
- Payload: `TaskCreatedPayload`

```go
type TaskCreatedPayload struct {
    TaskSlug      string
    CreatorUserID int64
    Title         string
    Category      string
    Region        string
    GoalSats      int64
}
```

Published when a new task campaign is created.

### DonationReceived

- Topic: `donation.received`
- Payload: `DonationReceivedPayload`

```go
type DonationReceivedPayload struct {
    TaskSlug   string
    AmountSats int64
    DonorID    int64
    Source     string
}
```

Represents an incoming funding donation for a task.

### ThresholdReached

- Topic: `threshold.reached`
- Payload: `ThresholdReachedPayload`

```go
type ThresholdReachedPayload struct {
    TaskSlug     string
    RequiredSigs int
    Signatures   int
}
```

Emitted when an escrow signature threshold is met.

### TaskLiquidating

- Topic: `task.liquidating`
- Payload: `TaskLiquidatingPayload`

```go
type TaskLiquidatingPayload struct {
    TaskSlug string
    Reason   string
}
```

Signals that a task has entered liquidation due to payout or financial state transition.

### TaskReadyForPayout

- Topic: `task.ready_for_payout`
- Payload: `TaskReadyForPayoutPayload`

```go
type TaskReadyForPayoutPayload struct {
    TaskSlug    string
    AmountSats  int64
    PayoutData  string
}
```

Indicates that a task is ready to execute a payout.

### TrusteeSigned

- Topic: `trustee.signed`
- Payload: `TrusteeSignedPayload`

```go
type TrusteeSignedPayload struct {
    TaskSlug            string
    TrusteePublicKeyHex string
    SignatureFragment   string
}
```

Published when a trustee signature fragment is collected.

### PayoutCompleted

- Topic: `payout.completed`
- Payload: `PayoutCompletedPayload`

```go
type PayoutCompletedPayload struct {
    TaskSlug      string
    L1TxID        string
    L2PaymentHash string
    TotalPaidSats int64
}
```

Represents the completion of a payout flow.

### TaskArchived

- Topic: `task.archived`
- Payload: `TaskArchivedPayload`

```go
type TaskArchivedPayload struct {
    TaskSlug string
    Reason   string
}
```

Indicates that a task has been archived.

### VolunteerVerified

- Topic: `volunteer.verified`
- Payload: not currently typed explicitly in `internal/events/events.go`

Represents volunteer verification status changes.

### ApplicationSubmitted

- Topic: `application.submitted`
- Payload: `ApplicationSubmittedPayload`

```go
type ApplicationSubmittedPayload struct {
    TaskSlug    string
    VolunteerID int64
}
```

Published when a volunteer applies to a task.

### SubmissionCreated

- Topic: `submission.created`
- Payload: `SubmissionCreatedPayload`

```go
type SubmissionCreatedPayload struct {
    TaskSlug    string
    VolunteerID int64
    Description string
}
```

Emitted when a volunteer submits work for a task.

### PaymentSettled

- Topic: `payment.settled`
- Payload: `PaymentSettledPayload`

```go
type PaymentSettledPayload struct {
    TaskSlug    string
    AmountSats  int64
    PaymentHash string
}
```

Indicates that an off-chain Lightning payment has settled.

### FinancialStateChanged

- Topic: `financial.state_changed`
- Payload: `FinancialStateChangedPayload`

```go
type FinancialStateChangedPayload struct {
    TaskSlug string
    OldState string
    NewState string
}
```

Published when a task's financial state transitions.

### TaskStatusChanged

- Topic: `task.status.changed`
- Payload: `TaskStatusChangedPayload`

```go
type TaskStatusChangedPayload struct {
    TaskSlug  string
    NewStatus string
}
```

Emitted for task lifecycle status transitions.

## Current Event Flow

Most event publishing and consumption is wired in `cmd/app/router.go` and service packages:

- `task` publishes `TaskCreated`, `TaskStatusChanged`, and `FinancialStateChanged`
- `volunteer` publishes `ApplicationSubmitted` and `SubmissionCreated`
- `escrow` publishes `ThresholdReached`
- `lightning` publishes `PaymentSettled`

The router currently subscribes to these events to update the ledger and trigger payout orchestration.

## Notes

- The event bus stores history and can replay events to new subscribers.
- Event subscribers should type-assert `Event.Payload` to the expected payload struct.
- Timestamp values are set automatically during publish.

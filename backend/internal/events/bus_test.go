package events

import (
    "sync"
    "testing"
    "time"
)

func TestPublishSubscribe(t *testing.T) {
    bus := NewEventBus()
    var wg sync.WaitGroup
    wg.Add(1)

    unsub := bus.Subscribe(TaskCreated, func(e Event) {
        if _, ok := e.Payload.(map[string]interface{}); !ok {
            t.Errorf("unexpected payload type")
        }
        wg.Done()
    })
    defer unsub()

    bus.Publish(Event{Type: TaskCreated, Payload: map[string]interface{}{"slug": "s1"}})
    done := make(chan struct{})
    go func() {
        wg.Wait()
        close(done)
    }()
    select {
    case <-done:
    case <-time.After(1 * time.Second):
        t.Fatal("timeout waiting for event")
    }
}

func TestMultipleSubscribersAndPanicRecovery(t *testing.T) {
    bus := NewEventBus()
    var wg sync.WaitGroup
    wg.Add(1)

    bus.Subscribe(DonationReceived, func(e Event) {
        wg.Done()
    })

    bus.Subscribe(DonationReceived, func(e Event) {
        panic("handler panic")
    })

    bus.Publish(Event{Type: DonationReceived, Payload: "x"})

    // second subscriber panics but first should still run
    done := make(chan struct{})
    go func() {
        wg.Wait()
        close(done)
    }()
    select {
    case <-done:
    case <-time.After(1 * time.Second):
        t.Fatal("timeout waiting for subscribers")
    }
}

func TestReplay(t *testing.T) {
    bus := NewEventBus()
    bus.Publish(Event{Type: ThresholdReached, Payload: ThresholdReachedPayload{TaskSlug: "task1"}})

    var wg sync.WaitGroup
    wg.Add(1)
    bus.Subscribe(ThresholdReached, func(e Event) {
        if payload, ok := e.Payload.(ThresholdReachedPayload); !ok || payload.TaskSlug != "task1" {
            t.Fatalf("unexpected replay payload: %#v", e.Payload)
        }
        wg.Done()
    })

    bus.Replay()
    done := make(chan struct{})
    go func() {
        wg.Wait()
        close(done)
    }()

    select {
    case <-done:
    case <-time.After(1 * time.Second):
        t.Fatal("timeout waiting for replay subscriber")
    }
}

func TestUnsubscribe(t *testing.T) {
    bus := NewEventBus()
    received := 0

    unsub := bus.Subscribe(TaskCreated, func(e Event) {
        received++
    })

    bus.Publish(Event{Type: TaskCreated, Payload: "first"})
    unsub()
    bus.Publish(Event{Type: TaskCreated, Payload: "second"})

    if received != 1 {
        t.Fatalf("expected 1 event after unsubscribe, got %d", received)
    }
}

func TestReplayPreservesPublishOrder(t *testing.T) {
    bus := NewEventBus()
    bus.Publish(Event{Type: ThresholdReached, Payload: ThresholdReachedPayload{TaskSlug: "task1", RequiredSigs: 1, Signatures: 1}})
    bus.Publish(Event{Type: ThresholdReached, Payload: ThresholdReachedPayload{TaskSlug: "task1", RequiredSigs: 2, Signatures: 2}})

    observed := make([]int, 0, 2)
    bus.Subscribe(ThresholdReached, func(e Event) {
        payload, ok := e.Payload.(ThresholdReachedPayload)
        if !ok {
            t.Fatalf("unexpected payload type: %#v", e.Payload)
        }
        observed = append(observed, payload.RequiredSigs)
    })

    bus.Replay()

    if len(observed) != 2 || observed[0] != 1 || observed[1] != 2 {
        t.Fatalf("expected replay order [1 2], got %v", observed)
    }
}

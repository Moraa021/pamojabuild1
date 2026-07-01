package events

import (
    "log"
    "sync"
    "time"
)

// EventBus is a simple in-memory pub/sub bus with replay ability.
type EventBus struct {
    mu          sync.RWMutex
    subscribers map[EventType]map[int]func(Event)
    nextID      int
    history     []Event
}

func NewEventBus() *EventBus {
    return &EventBus{
        subscribers: make(map[EventType]map[int]func(Event)),
        history:     make([]Event, 0),
    }
}

// Subscribe registers a handler for a given event type and returns an unsubscribe function.
func (b *EventBus) Subscribe(t EventType, handler func(Event)) func() {
    b.mu.Lock()
    defer b.mu.Unlock()
    if _, ok := b.subscribers[t]; !ok {
        b.subscribers[t] = make(map[int]func(Event))
    }
    id := b.nextID
    b.nextID++
    b.subscribers[t][id] = handler
    return func() {
        b.mu.Lock()
        defer b.mu.Unlock()
        delete(b.subscribers[t], id)
    }
}

// Publish sends the event to subscribers and appends to history for replay.
func (b *EventBus) Publish(e Event) {
    e.Timestamp = time.Now()
    // append to history
    b.mu.Lock()
    b.history = append(b.history, e)
    subs := make([]func(Event), 0)
    if handlers, ok := b.subscribers[e.Type]; ok {
        for _, h := range handlers {
            subs = append(subs, h)
        }
    }
    b.mu.Unlock()

    for _, h := range subs {
        func(h func(Event)) {
            defer func() {
                if r := recover(); r != nil {
                    log.Printf("event handler panic recovered: %v", r)
                }
            }()
            h(e)
        }(h)
    }
}

// Replay re-publishes events from history to current subscribers in order.
// Replay invokes handlers synchronously to ensure order and determinism
// when reapplying historical events (useful in tests/startup).
func (b *EventBus) Replay() {
    b.mu.RLock()
    hist := append([]Event(nil), b.history...)
    b.mu.RUnlock()

    for _, e := range hist {
        // invoke current subscribers synchronously with recovery
        b.mu.RLock()
        handlers := b.subscribers[e.Type]
        b.mu.RUnlock()

        for _, h := range handlers {
            func(h func(Event), ev Event) {
                defer func() {
                    if r := recover(); r != nil {
                        log.Printf("event handler panic recovered during replay: %v", r)
                    }
                }()
                h(ev)
            }(h, e)
        }
    }
}

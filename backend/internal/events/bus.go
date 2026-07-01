package events

import (
	"log"
	"sync"
)

type Handler func(event Event)

type EventBus struct {
	handlers map[EventType][]Handler
	mu       sync.RWMutex
}

func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[EventType][]Handler),
	}
}

func (eb *EventBus) Subscribe(eventType EventType, handler Handler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	
	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
	log.Printf("Subscribed to event: %s", eventType)
}

func (eb *EventBus) Publish(event Event) {
	eb.mu.RLock()
	handlers := eb.handlers[event.Type]
	eb.mu.RUnlock()
	
	log.Printf("Publishing event: %s", event.Type)
	for _, handler := range handlers {
		go func(h Handler) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Handler panic for event %s: %v", event.Type, r)
				}
			}()
			h(event)
		}(handler)
	}
}
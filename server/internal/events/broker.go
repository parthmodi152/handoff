// Package events provides an in-process pub/sub broker for request status changes.
package events

import (
	"encoding/json"
	"sync"
)

// Event represents a request status change.
type Event struct {
	RequestID string          `json:"request_id"`
	LoopID    string          `json:"loop_id"`
	Status    string          `json:"status"` // "completed", "cancelled", "timeout"
	Data      json.RawMessage `json:"data"`
}

// Broker is an in-process pub/sub for request events.
// Subscriptions are keyed by request ID — agents subscribe to exactly the request they created.
type Broker struct {
	mu   sync.Mutex
	subs map[string][]chan Event
}

// NewBroker creates a new event broker.
func NewBroker() *Broker {
	return &Broker{
		subs: make(map[string][]chan Event),
	}
}

// Subscribe registers a listener for events on a specific request ID.
// Returns a read-only channel and an unsubscribe function.
// The channel is buffered (size 1) since each request has exactly one terminal event.
func (b *Broker) Subscribe(requestID string) (<-chan Event, func()) {
	ch := make(chan Event, 1)

	b.mu.Lock()
	b.subs[requestID] = append(b.subs[requestID], ch)
	b.mu.Unlock()

	unsubscribe := func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		channels := b.subs[requestID]
		for i, c := range channels {
			if c == ch {
				b.subs[requestID] = append(channels[:i], channels[i+1:]...)
				break
			}
		}
		if len(b.subs[requestID]) == 0 {
			delete(b.subs, requestID)
		}
	}

	return ch, unsubscribe
}

// Publish sends an event to all subscribers for that request ID.
// Non-blocking: if a subscriber's channel is full, the event is dropped for that subscriber.
func (b *Broker) Publish(event Event) {
	b.mu.Lock()
	channels := make([]chan Event, len(b.subs[event.RequestID]))
	copy(channels, b.subs[event.RequestID])
	b.mu.Unlock()

	for _, ch := range channels {
		select {
		case ch <- event:
		default:
		}
	}
}

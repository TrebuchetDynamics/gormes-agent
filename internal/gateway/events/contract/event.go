package contract

import (
	"encoding/json"
	"time"
)

// Event is a typed message published on the event bus. Every event carries
// provenance (Source, TraceID) and a timestamp for ordering and debugging.
type Event struct {
	Type      string          `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Source    string          `json:"source"`
	Payload   json.RawMessage `json:"payload"`
	TraceID   string          `json:"trace_id"`
}

// EventHandler is a function that receives events from a subscribed topic.
type EventHandler func(Event)

// EventBus decouples publishers from subscribers via topic-based routing.
// Implementations must be safe for concurrent use.
type EventBus interface {
	// Publish sends an event to all subscribers of topic. Non-blocking —
	// slow consumers may drop events rather than block the publisher.
	Publish(topic string, event Event) error

	// Subscribe registers a handler for topic. Returns a function that
	// unsubscribes the handler.
	Subscribe(topic string, handler EventHandler) (unsubscribe func())

	// Close stops all subscriber goroutines and clears subscriptions.
	Close() error
}

// NewEvent creates an Event with the current UTC timestamp.
func NewEvent(typ, source string, payload json.RawMessage, traceID string) Event {
	return Event{
		Type:      typ,
		Timestamp: time.Now().UTC(),
		Source:    source,
		Payload:   payload,
		TraceID:   traceID,
	}
}

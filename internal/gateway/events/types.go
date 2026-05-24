package events

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

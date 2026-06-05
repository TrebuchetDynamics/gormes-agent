package events

import (
	"encoding/json"

	eventbus "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/events/bus"
)

// Event is a typed message published on the event bus. Every event carries
// provenance (Source, TraceID) and a timestamp for ordering and debugging.
type Event = eventbus.Event

// EventHandler is a function that receives events from a subscribed topic.
type EventHandler = eventbus.EventHandler

// NewEvent creates an Event with the current UTC timestamp.
func NewEvent(typ, source string, payload json.RawMessage, traceID string) Event {
	return eventbus.NewEvent(typ, source, payload, traceID)
}

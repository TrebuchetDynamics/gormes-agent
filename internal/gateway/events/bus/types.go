package bus

import (
	"encoding/json"

	eventcontract "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/events/contract"
)

// Event is a typed message published on the event bus. Every event carries
// provenance (Source, TraceID) and a timestamp for ordering and debugging.
type Event = eventcontract.Event

// EventHandler is a function that receives events from a subscribed topic.
type EventHandler = eventcontract.EventHandler

// NewEvent creates an Event with the current UTC timestamp.
func NewEvent(typ, source string, payload json.RawMessage, traceID string) Event {
	return eventcontract.NewEvent(typ, source, payload, traceID)
}

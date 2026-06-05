package adaptertest

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/events"
)

// MessageEventProbe captures gateway message-received events for adapter tests.
type MessageEventProbe struct {
	Dispatcher *gateway.EventDispatcher
	delivered  chan events.Event
}

// NewMessageEventProbe creates an in-process gateway event bus and subscribes
// to message-received events. The bus is closed during test cleanup.
func NewMessageEventProbe(t testing.TB) *MessageEventProbe {
	t.Helper()

	bus := events.NewInProcessEventBus()
	t.Cleanup(func() {
		_ = bus.Close()
	})

	probe := &MessageEventProbe{
		Dispatcher: gateway.NewEventDispatcher(bus),
		delivered:  make(chan events.Event, 1),
	}
	bus.Subscribe(gateway.TopicMessageReceived, func(e events.Event) {
		probe.delivered <- e
	})
	return probe
}

// Next waits for the next captured message event.
func (p *MessageEventProbe) Next(t testing.TB, label string) events.Event {
	t.Helper()

	select {
	case got := <-p.delivered:
		return got
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", label)
		return events.Event{}
	}
}

// RequireMessageReceived checks the event-bus envelope shared by channel bus
// adapter tests.
func RequireMessageReceived(t testing.TB, event events.Event, source, traceID string) {
	t.Helper()

	if event.Type != gateway.TopicMessageReceived {
		t.Fatalf("event type = %q, want %q", event.Type, gateway.TopicMessageReceived)
	}
	if event.Source != source || event.TraceID != traceID {
		t.Fatalf("event provenance = source:%q trace:%q, want %s/%s", event.Source, event.TraceID, source, traceID)
	}
}

// DecodeMessagePayload decodes a captured gateway message payload.
func DecodeMessagePayload(t testing.TB, event events.Event) gateway.MessageEventPayload {
	t.Helper()

	var payload gateway.MessageEventPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("payload decode: %v", err)
	}
	return payload
}

package busadapter

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestPublishInboundMessageNilReceiverReturnsUnavailable(t *testing.T) {
	var adapter *BusAdapter
	if err := adapter.PublishInboundMessage("", gateway.InboundEvent{}); err == nil {
		t.Fatalf("nil receiver error = nil, want unavailable error")
	}
}

func TestPublishInboundMessageMissingUnavailableErrorFallsBack(t *testing.T) {
	adapter := New(nil, "test", func(ev gateway.InboundEvent) string {
		return TraceIDFromChatMessage("test", ev)
	}, nil)
	if err := adapter.PublishInboundMessage("", gateway.InboundEvent{}); err == nil {
		t.Fatalf("missing unavailable error = nil, want fallback unavailable error")
	}
}

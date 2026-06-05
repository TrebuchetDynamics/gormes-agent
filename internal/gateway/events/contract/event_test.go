package contract

import (
	"encoding/json"
	"testing"
)

func TestNewEventCarriesProvenanceAndPayload(t *testing.T) {
	evt := NewEvent("gateway.message.received", "telegram", json.RawMessage(`{"text":"hi"}`), "trace-1")
	if evt.Type != "gateway.message.received" || evt.Source != "telegram" || evt.TraceID != "trace-1" {
		t.Fatalf("event provenance = %+v, want type/source/trace", evt)
	}
	if evt.Timestamp.IsZero() {
		t.Fatal("event timestamp is zero")
	}
	if string(evt.Payload) != `{"text":"hi"}` {
		t.Fatalf("payload = %s, want raw JSON preserved", evt.Payload)
	}
}

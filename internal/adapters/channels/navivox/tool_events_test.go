package navivox

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestNavivoxSendToolProgressStreamsUpdatedEvent(t *testing.T) {
	ch := newTestChannel(t)
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()
	conn := dialTestWebSocket(t, server.URL)
	defer conn.Close()

	if err := conn.WriteJSON(ClientMessage{Type: "subscribe_session", RequestID: "req-tool-update", SessionID: "s-tool-update"}); err != nil {
		t.Fatal(err)
	}
	var subscribed ServerEvent
	if err := conn.ReadJSON(&subscribed); err != nil {
		t.Fatal(err)
	}
	if subscribed.Type != "session_started" || subscribed.RequestID != "req-tool-update" || subscribed.SessionID != "s-tool-update" {
		t.Fatalf("session_started = %+v", subscribed)
	}

	msgID, err := ch.SendToolProgress(context.Background(), "s-tool-update", gateway.ToolProgressEvent{
		ID:       "call-browser",
		ToolName: "browser_navigate",
		Status:   gateway.ToolProgressUpdated,
		Summary:  "browser_navigate updated with redacted progress",
		Metadata: map[string]any{
			"artifact_ref": "artifact://browser-state",
			"secret_token": "must-not-leak",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if msgID != "call-browser" {
		t.Fatalf("SendToolProgress msgID = %q, want call-browser", msgID)
	}

	var event ServerEvent
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatal(err)
	}
	if event.Type != "tool_call_updated" || event.RequestID != "req-tool-update" || event.SessionID != "s-tool-update" {
		t.Fatalf("tool update event envelope = %+v", event)
	}
	if event.ToolCallID != "call-browser" || event.ToolName != "browser_navigate" || event.Status != "updated" {
		t.Fatalf("tool update event fields = %+v", event)
	}
	if event.Message != "browser_navigate updated with redacted progress" {
		t.Fatalf("tool update summary = %q", event.Message)
	}
	if event.Metadata["artifact_ref"] != "artifact://browser-state" {
		t.Fatalf("tool update metadata = %+v", event.Metadata)
	}
	if _, leaked := event.Metadata["secret_token"]; leaked {
		t.Fatalf("tool update metadata leaked secret: %+v", event.Metadata)
	}
}

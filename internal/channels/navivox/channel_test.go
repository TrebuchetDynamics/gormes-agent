package navivox

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestStatusRequiresAuthAndHealthzIsPublic(t *testing.T) {
	ch := newTestChannel(t)
	inbox := make(chan gateway.InboundEvent, 1)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()

	healthResp, err := http.Get(server.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer healthResp.Body.Close()
	if healthResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /healthz status = %d, want 200", healthResp.StatusCode)
	}

	statusResp, err := http.Get(server.URL + "/v1/navivox/status")
	if err != nil {
		t.Fatal(err)
	}
	defer statusResp.Body.Close()
	if statusResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d, want 401", statusResp.StatusCode)
	}
}

func TestHTTPStartTurnRequiresAuthAndEnqueuesTypedGatewayEvent(t *testing.T) {
	ch := newTestChannel(t)
	inbox := make(chan gateway.InboundEvent, 1)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()

	unauthResp, err := http.Post(server.URL+"/v1/navivox/turn", "application/json", strings.NewReader(`{"request_id":"req-1","text":"hello"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer unauthResp.Body.Close()
	if unauthResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized turn status = %d, want 401", unauthResp.StatusCode)
	}

	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/navivox/turn", strings.NewReader(`{"request_id":"req-1","session_id":"s-1","text":"hello from navivox"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer nvbx_test_token")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("authorized turn status = %d, want 202", resp.StatusCode)
	}

	select {
	case ev := <-inbox:
		if ev.Platform != PlatformName || ev.Kind != gateway.EventSubmit || ev.ChatID != "s-1" || ev.UserID != "navivox" || ev.MsgID != "req-1" {
			t.Fatalf("gateway event = %+v, want navivox submit for s-1/req-1", ev)
		}
		if ev.Text != "hello from navivox" {
			t.Fatalf("gateway event text = %q", ev.Text)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for gateway event")
	}
}

func TestWebSocketPingAndMalformedJSONReturnTypedEvents(t *testing.T) {
	ch := newTestChannel(t)
	inbox := make(chan gateway.InboundEvent, 1)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()
	conn := dialTestWebSocket(t, server.URL)
	defer conn.Close()

	if err := conn.WriteJSON(ClientMessage{Type: "ping", RequestID: "req-ping"}); err != nil {
		t.Fatal(err)
	}
	var pong ServerEvent
	if err := conn.ReadJSON(&pong); err != nil {
		t.Fatal(err)
	}
	if pong.Type != "pong" || pong.RequestID != "req-ping" {
		t.Fatalf("pong = %+v, want request_id=req-ping", pong)
	}

	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{bad json`)); err != nil {
		t.Fatal(err)
	}
	var event ServerEvent
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatal(err)
	}
	if event.Type != "error" || event.Code != "bad_request" {
		t.Fatalf("malformed JSON event = %+v, want bad_request error", event)
	}
}

func TestWebSocketStartTurnStreamsGatewayResponses(t *testing.T) {
	ch := newTestChannel(t)
	inbox := make(chan gateway.InboundEvent, 1)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()
	conn := dialTestWebSocket(t, server.URL)
	defer conn.Close()

	if err := conn.WriteJSON(ClientMessage{Type: "start_turn", RequestID: "req-turn", SessionID: "s-2", Text: "stream this"}); err != nil {
		t.Fatal(err)
	}
	var started ServerEvent
	if err := conn.ReadJSON(&started); err != nil {
		t.Fatal(err)
	}
	if started.Type != "session_started" || started.RequestID != "req-turn" || started.SessionID != "s-2" {
		t.Fatalf("session_started = %+v", started)
	}

	select {
	case ev := <-inbox:
		if ev.Kind != gateway.EventSubmit || ev.ChatID != "s-2" || ev.Text != "stream this" {
			t.Fatalf("gateway event = %+v, want submit for s-2", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for gateway event")
	}

	msgID, err := ch.SendPlaceholder(context.Background(), "s-2")
	if err != nil {
		t.Fatal(err)
	}
	if err := ch.EditMessage(context.Background(), "s-2", msgID, "hel"); err != nil {
		t.Fatal(err)
	}
	if err := ch.EditMessage(context.Background(), "s-2", msgID, "hello"); err != nil {
		t.Fatal(err)
	}
	if err := ch.EditMessageFinal(context.Background(), "s-2", msgID, "hello", true); err != nil {
		t.Fatal(err)
	}

	var delta1 ServerEvent
	if err := conn.ReadJSON(&delta1); err != nil {
		t.Fatal(err)
	}
	if delta1.Type != "assistant_delta" || delta1.RequestID != "req-turn" || delta1.Text != "hel" {
		t.Fatalf("first delta = %+v", delta1)
	}
	var delta2 ServerEvent
	if err := conn.ReadJSON(&delta2); err != nil {
		t.Fatal(err)
	}
	if delta2.Type != "assistant_delta" || delta2.Text != "lo" {
		t.Fatalf("second delta = %+v", delta2)
	}
	var final ServerEvent
	if err := conn.ReadJSON(&final); err != nil {
		t.Fatal(err)
	}
	if final.Type != "assistant_message" || final.Text != "hello" {
		t.Fatalf("assistant_message = %+v", final)
	}
	var done ServerEvent
	if err := conn.ReadJSON(&done); err != nil {
		t.Fatal(err)
	}
	if done.Type != "done" || done.SessionID != "s-2" {
		t.Fatalf("done = %+v", done)
	}
}

func newTestChannel(t *testing.T) *Channel {
	t.Helper()
	ch, err := NewChannel(config.NavivoxCfg{
		Enabled:      true,
		BindHost:     config.NavivoxDefaultBindHost,
		Port:         config.NavivoxDefaultPort,
		ExposureMode: config.NavivoxExposureLocal,
		AuthMode:     config.NavivoxAuthPairingToken,
		Token:        "nvbx_test_token",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	ch.newID = func() string { return "generated-id" }
	return ch
}

func dialTestWebSocket(t *testing.T, httpURL string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(httpURL, "http") + "/v1/navivox/stream"
	header := http.Header{}
	header.Set("Authorization", "Bearer nvbx_test_token")
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		if resp != nil {
			t.Fatalf("websocket dial status=%d err=%v", resp.StatusCode, err)
		}
		t.Fatal(err)
	}
	return conn
}

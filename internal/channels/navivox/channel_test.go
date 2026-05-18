package navivox

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/network/vpnhost"
)

func TestNavivoxStatusRequiresAuthAndHealthzIsPublic(t *testing.T) {
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

func TestNavivoxHTTPStartTurnRequiresAuthAndEnqueuesTypedGatewayEvent(t *testing.T) {
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

func TestNavivoxWebSocketAuthAcceptsBrowserSubprotocolToken(t *testing.T) {
	ch := newTestChannel(t)
	inbox := make(chan gateway.InboundEvent, 1)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/v1/navivox/stream"
	dialer := websocket.Dialer{
		Subprotocols: []string{
			navivoxWebSocketProtocol,
			navivoxWebSocketTokenProtocolPrefix + base64.RawURLEncoding.EncodeToString([]byte("nvbx_test_token")),
		},
	}
	conn, resp, err := dialer.Dial(wsURL, nil)
	if err != nil {
		if resp != nil {
			t.Fatalf("websocket dial status=%d err=%v", resp.StatusCode, err)
		}
		t.Fatal(err)
	}
	defer conn.Close()
	if conn.Subprotocol() != navivoxWebSocketProtocol {
		t.Fatalf("selected websocket subprotocol = %q, want %q", conn.Subprotocol(), navivoxWebSocketProtocol)
	}

	if err := conn.WriteJSON(ClientMessage{Type: "ping", RequestID: "req-browser"}); err != nil {
		t.Fatal(err)
	}
	var pong ServerEvent
	if err := conn.ReadJSON(&pong); err != nil {
		t.Fatal(err)
	}
	if pong.Type != "pong" || pong.RequestID != "req-browser" {
		t.Fatalf("pong = %+v, want browser-authenticated pong", pong)
	}
}

func TestNavivoxWebSocketAuthRejectsBadBrowserSubprotocolToken(t *testing.T) {
	ch := newTestChannel(t)
	inbox := make(chan gateway.InboundEvent, 1)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/v1/navivox/stream"
	dialer := websocket.Dialer{
		Subprotocols: []string{
			navivoxWebSocketProtocol,
			navivoxWebSocketTokenProtocolPrefix + base64.RawURLEncoding.EncodeToString([]byte("wrong")),
		},
	}
	conn, resp, err := dialer.Dial(wsURL, nil)
	if err == nil {
		conn.Close()
		t.Fatal("websocket dial with bad subprotocol token succeeded, want 401")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		if resp == nil {
			t.Fatalf("websocket dial response = nil err=%v, want 401", err)
		}
		t.Fatalf("websocket dial status=%d err=%v, want 401", resp.StatusCode, err)
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

func TestNavivoxWebSocketStartTurnStreamsGatewayResponses(t *testing.T) {
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

	delta1 := readNonProfileContactEvent(t, conn)
	if delta1.Type != "assistant_delta" || delta1.RequestID != "req-turn" || delta1.Text != "hel" {
		t.Fatalf("first delta = %+v", delta1)
	}
	delta2 := readNonProfileContactEvent(t, conn)
	if delta2.Type != "assistant_delta" || delta2.Text != "lo" {
		t.Fatalf("second delta = %+v", delta2)
	}
	final := readNonProfileContactEvent(t, conn)
	if final.Type != "assistant_message" || final.Text != "hello" {
		t.Fatalf("assistant_message = %+v", final)
	}
	done := readNonProfileContactEvent(t, conn)
	if done.Type != "done" || done.SessionID != "s-2" {
		t.Fatalf("done = %+v", done)
	}
}

func TestNavivoxHTTPStartTurnStreamsToSubscribedWebSocket(t *testing.T) {
	ch := newTestChannel(t)
	inbox := make(chan gateway.InboundEvent, 1)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()
	conn := dialTestWebSocket(t, server.URL)
	defer conn.Close()

	if err := conn.WriteJSON(ClientMessage{Type: "subscribe_session", RequestID: "req-http", SessionID: "s-http"}); err != nil {
		t.Fatal(err)
	}
	var subscribed ServerEvent
	if err := conn.ReadJSON(&subscribed); err != nil {
		t.Fatal(err)
	}
	if subscribed.Type != "session_started" || subscribed.RequestID != "req-http" || subscribed.SessionID != "s-http" {
		t.Fatalf("session_started = %+v", subscribed)
	}

	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/navivox/turn", strings.NewReader(`{"request_id":"req-http","session_id":"s-http","text":"posted turn"}`))
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
		if ev.Kind != gateway.EventSubmit || ev.ChatID != "s-http" || ev.Text != "posted turn" {
			t.Fatalf("gateway event = %+v, want HTTP submit for s-http", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for HTTP gateway event")
	}

	msgID, err := ch.SendPlaceholder(context.Background(), "s-http")
	if err != nil {
		t.Fatal(err)
	}
	if err := ch.EditMessage(context.Background(), "s-http", msgID, "hello"); err != nil {
		t.Fatal(err)
	}
	if err := ch.EditMessageFinal(context.Background(), "s-http", msgID, "hello posted turn", true); err != nil {
		t.Fatal(err)
	}

	delta := readNonProfileContactEvent(t, conn)
	if delta.Type != "assistant_delta" || delta.RequestID != "req-http" || delta.Text != "hello" {
		t.Fatalf("delta = %+v", delta)
	}
	final := readNonProfileContactEvent(t, conn)
	if final.Type != "assistant_message" || final.RequestID != "req-http" || final.Text != "hello posted turn" {
		t.Fatalf("assistant_message = %+v", final)
	}
	done := readNonProfileContactEvent(t, conn)
	if done.Type != "done" || done.SessionID != "s-http" {
		t.Fatalf("done = %+v", done)
	}
}

func TestNavivoxSendToolProgressStreamsStructuredToolEvent(t *testing.T) {
	ch := newTestChannel(t)
	inbox := make(chan gateway.InboundEvent, 1)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()
	conn := dialTestWebSocket(t, server.URL)
	defer conn.Close()

	if err := conn.WriteJSON(ClientMessage{Type: "subscribe_session", RequestID: "req-tool", SessionID: "s-tool"}); err != nil {
		t.Fatal(err)
	}
	var subscribed ServerEvent
	if err := conn.ReadJSON(&subscribed); err != nil {
		t.Fatal(err)
	}
	if subscribed.Type != "session_started" || subscribed.RequestID != "req-tool" || subscribed.SessionID != "s-tool" {
		t.Fatalf("session_started = %+v", subscribed)
	}

	msgID, err := ch.SendToolProgress(context.Background(), "s-tool", gateway.ToolProgressEvent{
		ID:       "call-browser",
		ToolName: "browser_navigate",
		Status:   gateway.ToolProgressStarted,
		Summary:  "browser_navigate started with redacted input",
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
	if event.Type != "tool_call_started" || event.RequestID != "req-tool" || event.SessionID != "s-tool" {
		t.Fatalf("tool event envelope = %+v", event)
	}
	if event.ToolCallID != "call-browser" || event.ToolName != "browser_navigate" || event.Status != "started" {
		t.Fatalf("tool event fields = %+v", event)
	}
	if event.Message != "browser_navigate started with redacted input" {
		t.Fatalf("tool event summary = %q", event.Message)
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

func readNonProfileContactEvent(t *testing.T, conn *websocket.Conn) ServerEvent {
	t.Helper()
	for i := 0; i < 8; i++ {
		var event ServerEvent
		if err := conn.ReadJSON(&event); err != nil {
			t.Fatal(err)
		}
		if event.Type != "profile_contact_update" {
			return event
		}
	}
	t.Fatal("only received profile_contact_update events")
	return ServerEvent{}
}

func TestNewChannel_TailscaleExposureWithLoopbackBind_FailsClosed(t *testing.T) {
	prev := vpnHostLister
	t.Cleanup(func() { vpnHostLister = prev })
	vpnHostLister = func(context.Context) ([]vpnhost.Host, error) {
		return []vpnhost.Host{
			{Iface: "tailscale0", Kind: vpnhost.KindTailscale, IPv4: "100.64.1.2"},
		}, nil
	}

	_, err := NewChannel(config.NavivoxCfg{
		Enabled:      true,
		BindHost:     "127.0.0.1",
		Port:         config.NavivoxDefaultPort,
		ExposureMode: config.NavivoxExposureTailscale,
		AuthMode:     config.NavivoxAuthStaticToken,
		Token:        "x",
	}, nil)
	if err == nil {
		t.Fatal("NewChannel err = nil, want VPN bind mismatch error")
	}
	for _, want := range []string{"127.0.0.1", "100.64.1.2", "exposure_mode=tailscale"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, missing %q", err, want)
		}
	}
}

func TestNewChannel_TailscaleExposureWithMatchingVPNBind_Succeeds(t *testing.T) {
	prev := vpnHostLister
	t.Cleanup(func() { vpnHostLister = prev })
	vpnHostLister = func(context.Context) ([]vpnhost.Host, error) {
		return []vpnhost.Host{
			{Iface: "tailscale0", Kind: vpnhost.KindTailscale, IPv4: "100.64.1.2"},
		}, nil
	}

	ch, err := NewChannel(config.NavivoxCfg{
		Enabled:      true,
		BindHost:     "100.64.1.2",
		Port:         config.NavivoxDefaultPort,
		ExposureMode: config.NavivoxExposureTailscale,
		AuthMode:     config.NavivoxAuthStaticToken,
		Token:        "x",
	}, nil)
	if err != nil {
		t.Fatalf("NewChannel err = %v, want nil for matching VPN bind", err)
	}
	if ch == nil {
		t.Fatal("NewChannel returned nil channel")
	}
}

func TestNewChannel_LocalExposureUnaffectedByVPNCheck(t *testing.T) {
	prev := vpnHostLister
	t.Cleanup(func() { vpnHostLister = prev })
	vpnHostLister = func(context.Context) ([]vpnhost.Host, error) {
		return nil, nil // no VPN
	}

	ch, err := NewChannel(config.NavivoxCfg{
		Enabled:      true,
		BindHost:     "127.0.0.1",
		Port:         config.NavivoxDefaultPort,
		ExposureMode: config.NavivoxExposureLocal,
		AuthMode:     config.NavivoxAuthStaticToken,
		Token:        "x",
	}, nil)
	if err != nil {
		t.Fatalf("NewChannel err = %v, want nil (local mode is not VPN-gated)", err)
	}
	if ch == nil {
		t.Fatal("NewChannel returned nil channel")
	}
}

package navivox

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
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

	req, err := http.NewRequest(http.MethodGet, server.URL+"/v1/navivox/status", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer nvbx_test_token")
	authStatus, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer authStatus.Body.Close()
	if authStatus.StatusCode != http.StatusOK {
		t.Fatalf("authorized status = %d, want 200", authStatus.StatusCode)
	}
	var payload map[string]any
	if err := json.NewDecoder(authStatus.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload["protocol_version"] != "navivox.v1" {
		t.Fatalf("protocol_version = %v, want navivox.v1", payload["protocol_version"])
	}
	protocols, ok := payload["websocket_protocols"].([]any)
	if !ok || len(protocols) < 2 || protocols[0] != "navivox.v1" || protocols[1] != "gormes.navivox.v1" {
		t.Fatalf("websocket_protocols = %#v, want neutral protocol plus legacy gormes fallback", payload["websocket_protocols"])
	}
	capabilities, ok := payload["capabilities"].([]any)
	if !ok || !containsAny(capabilities, "profile_contacts") || !containsAny(capabilities, "turn_control") {
		t.Fatalf("capabilities = %#v, want profile_contacts and turn_control", payload["capabilities"])
	}
}

func TestNavivoxStatusIncludesSetupHandoffForAppContinuation(t *testing.T) {
	ch := newTestChannel(t)
	inbox := make(chan gateway.InboundEvent, 1)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/v1/navivox/status", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer nvbx_test_token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	capabilities, ok := payload["capabilities"].([]any)
	if !ok || !containsAny(capabilities, "setup_handoff") {
		t.Fatalf("capabilities = %#v, want setup_handoff", payload["capabilities"])
	}
	handoff, ok := payload["setup_handoff"].(map[string]any)
	if !ok {
		t.Fatalf("setup_handoff = %#v, want object", payload["setup_handoff"])
	}
	if handoff["recommended_path"] != "navivox" {
		t.Fatalf("recommended_path = %#v, want navivox", handoff["recommended_path"])
	}
	if handoff["title"] != "Continue setup in Navivox" {
		t.Fatalf("title = %#v", handoff["title"])
	}
	steps, ok := handoff["steps"].([]any)
	if !ok || len(steps) < 4 {
		t.Fatalf("steps = %#v, want provider/model/workspace/channel setup steps", handoff["steps"])
	}
	if handoff["mutation_policy"] != "read_only_handoff" {
		t.Fatalf("mutation_policy = %#v, want read_only_handoff", handoff["mutation_policy"])
	}
	if handoff["entry_screen"] != "setup.provider" {
		t.Fatalf("entry_screen = %#v, want setup.provider", handoff["entry_screen"])
	}
	sections, ok := handoff["sections"].([]any)
	if !ok || len(sections) != 4 {
		t.Fatalf("sections = %#v, want four structured setup sections", handoff["sections"])
	}
	for i, want := range []struct {
		id      string
		title   string
		screen  string
		command string
	}{
		{"provider", "Choose provider", "setup.provider", "gormes setup provider"},
		{"model", "Choose model", "setup.model", "gormes setup model"},
		{"workspace", "Confirm workspace", "setup.workspace", "gormes setup workspace"},
		{"channels", "Enable channels", "setup.channels", "gormes setup gateway"},
	} {
		section, ok := sections[i].(map[string]any)
		if !ok {
			t.Fatalf("section[%d] = %#v, want object", i, sections[i])
		}
		if section["id"] != want.id || section["title"] != want.title || section["navivox_screen"] != want.screen || section["fallback_cli_command"] != want.command {
			t.Fatalf("section[%d] = %#v, want id=%s title=%q screen=%s fallback=%q", i, section, want.id, want.title, want.screen, want.command)
		}
	}
	for _, secretLike := range []string{"api_key", "token", "secret", "password"} {
		if strings.Contains(strings.ToLower(fmt.Sprint(handoff)), secretLike) {
			t.Fatalf("setup handoff must not expose secret fields: %#v", handoff)
		}
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

func TestNavivoxLayeredAuthRequiresTokenAndAllowedTailscaleIdentity(t *testing.T) {
	prev := vpnHostLister
	t.Cleanup(func() { vpnHostLister = prev })
	vpnHostLister = func(context.Context) ([]vpnhost.Host, error) {
		return []vpnhost.Host{{Iface: "tailscale0", Kind: vpnhost.KindTailscale, IPv4: "100.64.1.2"}}, nil
	}

	ch, err := NewChannel(config.NavivoxCfg{
		Enabled:                  true,
		BindHost:                 "100.64.1.2",
		Port:                     config.NavivoxDefaultPort,
		ExposureMode:             config.NavivoxExposureTailscale,
		AuthMode:                 config.NavivoxAuthTokenAndTailscaleIdentity,
		Token:                    "nvbx_test_token",
		AllowedTailnetIdentities: []string{"juan@example.com"},
		AllowOrigins:             []string{"*"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	inbox := make(chan gateway.InboundEvent, 1)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()

	for name, headers := range map[string]map[string]string{
		"token-only": {
			"Authorization": "Bearer nvbx_test_token",
		},
		"identity-only": {
			"Tailscale-User-Login": "juan@example.com",
		},
		"wrong-identity": {
			"Authorization":        "Bearer nvbx_test_token",
			"Tailscale-User-Login": "intruder@example.com",
		},
	} {
		t.Run(name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, server.URL+"/v1/navivox/status", nil)
			if err != nil {
				t.Fatal(err)
			}
			for key, value := range headers {
				req.Header.Set(key, value)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", resp.StatusCode)
			}
		})
	}

	req, err := http.NewRequest(http.MethodGet, server.URL+"/v1/navivox/status", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer nvbx_test_token")
	req.Header.Set("Tailscale-User-Login", "juan@example.com")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
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

func TestNavivoxVoiceMarkedStreamTurnEnqueuesTranscriptOnlyGatewayEvent(t *testing.T) {
	ch := newTestChannel(t)
	inbox := make(chan gateway.InboundEvent, 1)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()
	conn := dialTestWebSocket(t, server.URL)
	defer conn.Close()

	if err := conn.WriteJSON(ClientMessage{
		Type:      "start_turn",
		RequestID: "req-voice",
		SessionID: "s-voice",
		Text:      "transcribed voice command",
		Metadata: map[string]any{
			"input_kind":   "voice",
			"stt_evidence": "device_transcribed",
			"audio_path":   "/home/xel/private/raw-audio.wav",
		},
	}); err != nil {
		t.Fatal(err)
	}
	var started ServerEvent
	if err := conn.ReadJSON(&started); err != nil {
		t.Fatal(err)
	}
	if started.Type != "session_started" || started.RequestID != "req-voice" || started.SessionID != "s-voice" {
		t.Fatalf("session_started = %+v", started)
	}

	select {
	case ev := <-inbox:
		if ev.Kind != gateway.EventSubmit || ev.ChatID != "s-voice" || ev.Text != "transcribed voice command" {
			t.Fatalf("gateway event = %+v, want transcript-only submit for s-voice", ev)
		}
		if len(ev.Attachments) != 0 {
			t.Fatalf("gateway attachments = %+v, want no raw audio attached by Navivox channel", ev.Attachments)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for voice-marked gateway event")
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

func TestNavivoxWebSocketStopTurnEnqueuesCancel(t *testing.T) {
	ch := newTestChannel(t)
	inbox := make(chan gateway.InboundEvent, 1)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()
	conn := dialTestWebSocket(t, server.URL)
	defer conn.Close()

	if err := conn.WriteJSON(ClientMessage{Type: "stop_turn", RequestID: "req-stop", SessionID: "s-stop"}); err != nil {
		t.Fatal(err)
	}

	var done ServerEvent
	if err := conn.ReadJSON(&done); err != nil {
		t.Fatal(err)
	}
	if done.Type != "done" || done.RequestID != "req-stop" || done.SessionID != "s-stop" || done.Status != "stopped" {
		t.Fatalf("stop response = %+v", done)
	}

	select {
	case ev := <-inbox:
		if ev.Kind != gateway.EventCancel || ev.ChatID != "s-stop" {
			t.Fatalf("gateway event = %+v, want cancel for s-stop", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for stop gateway event")
	}
}

func TestNavivoxSafetyEventsStreamAsTypedEvents(t *testing.T) {
	ch := newTestChannel(t)
	inbox := make(chan gateway.InboundEvent, 1)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()
	conn := dialTestWebSocket(t, server.URL)
	defer conn.Close()

	if err := conn.WriteJSON(ClientMessage{Type: "subscribe_session", RequestID: "req-safe", SessionID: "s-safe"}); err != nil {
		t.Fatal(err)
	}
	var subscribed ServerEvent
	if err := conn.ReadJSON(&subscribed); err != nil {
		t.Fatal(err)
	}
	if subscribed.Type != "session_started" || subscribed.SessionID != "s-safe" {
		t.Fatalf("session_started = %+v", subscribed)
	}

	warningID, err := ch.SendSafetyWarning(context.Background(), "s-safe", SafetyEvent{
		ID:       "safe-1",
		Severity: "high",
		Message:  "Shell command wants to modify files",
		Risk:     "Writes may change the workspace",
	})
	if err != nil {
		t.Fatal(err)
	}
	if warningID != "safe-1" {
		t.Fatalf("warning id = %q, want safe-1", warningID)
	}
	var warning ServerEvent
	if err := conn.ReadJSON(&warning); err != nil {
		t.Fatal(err)
	}
	if warning.Type != "safety_warning" || warning.RequestID != "req-safe" || warning.SessionID != "s-safe" {
		t.Fatalf("warning envelope = %+v", warning)
	}
	if warning.SafetyID != "safe-1" || warning.Severity != "high" || warning.Message != "Shell command wants to modify files" || warning.Risk != "Writes may change the workspace" {
		t.Fatalf("warning fields = %+v", warning)
	}

	approvalID, err := ch.SendApprovalRequired(context.Background(), "s-safe", ApprovalEvent{
		ID:         "approval-1",
		ToolCallID: "call-shell",
		Prompt:     "Approve shell.run?",
		Risk:       "Command can edit files",
	})
	if err != nil {
		t.Fatal(err)
	}
	if approvalID != "approval-1" {
		t.Fatalf("approval id = %q, want approval-1", approvalID)
	}
	var approval ServerEvent
	if err := conn.ReadJSON(&approval); err != nil {
		t.Fatal(err)
	}
	if approval.Type != "approval_required" || approval.ApprovalID != "approval-1" || approval.ToolCallID != "call-shell" || approval.Message != "Approve shell.run?" || approval.Risk != "Command can edit files" {
		t.Fatalf("approval event = %+v", approval)
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

func containsAny(values []any, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
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
		AllowOrigins: []string{"*"},
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

func TestMergeProfileContact_PreservesLoaderHealth(t *testing.T) {
	base := ProfileContact{
		ServerID:         "navivox-gateway",
		ProfileID:        "default",
		DisplayName:      "Default profile",
		Health:           ProfileContactHealthWarning,
		AttentionBadges:  []string{"config", "workspace"},
		WorkspaceRootsOK: false,
	}
	overlay := ProfileContact{
		ServerID:        "navivox-gateway",
		ProfileID:       "default",
		LatestPreview:   "active turn",
		ActiveTurnState: ProfileContactTurnActive,
		Health:          ProfileContactHealthOnline,
		MicAvailable:    true,
	}
	merged := mergeProfileContact(base, overlay)
	if merged.Health != ProfileContactHealthWarning {
		t.Fatalf("merged health = %q, want %q (loader health preserved)", merged.Health, ProfileContactHealthWarning)
	}
	if merged.LatestPreview != "active turn" {
		t.Fatalf("merged preview = %q, want active turn", merged.LatestPreview)
	}
	if merged.ActiveTurnState != ProfileContactTurnActive {
		t.Fatalf("merged turn state = %q, want active", merged.ActiveTurnState)
	}
}

func TestNavivoxCORSPreflightAndActualRequest(t *testing.T) {
	ch := newTestChannel(t)
	inbox := make(chan gateway.InboundEvent, 1)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()

	origin := "http://localhost:3000"
	preflight, err := http.NewRequest(http.MethodOptions, server.URL+"/v1/navivox/status", nil)
	if err != nil {
		t.Fatal(err)
	}
	preflight.Header.Set("Origin", origin)
	preflightResp, err := http.DefaultClient.Do(preflight)
	if err != nil {
		t.Fatal(err)
	}
	defer preflightResp.Body.Close()
	if preflightResp.StatusCode != http.StatusNoContent {
		t.Fatalf("CORS preflight status = %d, want 204", preflightResp.StatusCode)
	}
	if got := preflightResp.Header.Get("Access-Control-Allow-Origin"); got != origin {
		t.Fatalf("CORS Allow-Origin = %q, want %q", got, origin)
	}

	actual, err := http.NewRequest(http.MethodGet, server.URL+"/healthz", nil)
	if err != nil {
		t.Fatal(err)
	}
	actual.Header.Set("Origin", origin)
	actualResp, err := http.DefaultClient.Do(actual)
	if err != nil {
		t.Fatal(err)
	}
	defer actualResp.Body.Close()
	if got := actualResp.Header.Get("Access-Control-Allow-Origin"); got != origin {
		t.Fatalf("CORS Allow-Origin on actual request = %q, want %q", got, origin)
	}
	if got := actualResp.Header.Get("Vary"); got != "Origin" {
		t.Fatalf("CORS Vary = %q, want Origin", got)
	}
}

func TestNavivoxWebSocketCancelTurnEnqueuesCancel(t *testing.T) {
	ch := newTestChannel(t)
	inbox := make(chan gateway.InboundEvent, 1)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()
	conn := dialTestWebSocket(t, server.URL)
	defer conn.Close()

	if err := conn.WriteJSON(ClientMessage{Type: "cancel_turn", RequestID: "req-cancel", SessionID: "s-cancel"}); err != nil {
		t.Fatal(err)
	}

	var done ServerEvent
	if err := conn.ReadJSON(&done); err != nil {
		t.Fatal(err)
	}
	if done.Type != "done" || done.RequestID != "req-cancel" || done.SessionID != "s-cancel" || done.Status != "cancelled" {
		t.Fatalf("cancel response = %+v", done)
	}

	select {
	case ev := <-inbox:
		if ev.Kind != gateway.EventCancel || ev.ChatID != "s-cancel" {
			t.Fatalf("gateway event = %+v, want cancel for s-cancel", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for cancel gateway event")
	}
}

func TestNavivoxSubscriberLifecycleAndBroadcast(t *testing.T) {
	ch := newTestChannel(t)
	inbox := make(chan gateway.InboundEvent, 1)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()

	conn1 := dialTestWebSocket(t, server.URL)
	defer conn1.Close()

	if err := conn1.WriteJSON(ClientMessage{Type: "subscribe_session", RequestID: "req-sub", SessionID: "s-sub"}); err != nil {
		t.Fatal(err)
	}
	var subscribed ServerEvent
	if err := conn1.ReadJSON(&subscribed); err != nil {
		t.Fatal(err)
	}
	if subscribed.Type != "session_started" || subscribed.SessionID != "s-sub" {
		t.Fatalf("subscribe response = %+v", subscribed)
	}

	ch.mu.Lock()
	state := ch.sessions["s-sub"]
	if state == nil {
		t.Fatal("session not created after subscribe")
	}
	if state.Subscribers != 1 {
		t.Fatalf("subscribers = %d, want 1 after first subscribe", state.Subscribers)
	}
	ch.mu.Unlock()

	if err := conn1.WriteJSON(ClientMessage{Type: "subscribe_session", RequestID: "req-sub2", SessionID: "s-sub"}); err != nil {
		t.Fatal(err)
	}
	var subscribed2 ServerEvent
	if err := conn1.ReadJSON(&subscribed2); err != nil {
		t.Fatal(err)
	}
	if subscribed2.Type != "session_started" {
		t.Fatalf("second subscribe response = %+v", subscribed2)
	}

	ch.mu.Lock()
	state = ch.sessions["s-sub"]
	if state.Subscribers != 1 {
		t.Fatalf("subscribers = %d, want 1 (deduplicated re-subscribe)", state.Subscribers)
	}
	ch.mu.Unlock()

	conn1.Close()
	time.Sleep(100 * time.Millisecond)

	ch.mu.Lock()
	state = ch.sessions["s-sub"]
	if state.Subscribers != 0 {
		t.Fatalf("subscribers after disconnect = %d, want 0", state.Subscribers)
	}
	ch.mu.Unlock()
}

func TestNavivoxSessionSweepEvictsOldSessions(t *testing.T) {
	ch := newTestChannel(t)
	ch.now = func() time.Time { return time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC) }

	ch.mu.Lock()
	ch.sessions["old-session"] = &sessionState{
		ID:        "old-session",
		CreatedAt: time.Date(2024, 12, 30, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2024, 12, 30, 12, 0, 0, 0, time.UTC),
	}
	ch.sessions["fresh-session"] = &sessionState{
		ID:        "fresh-session",
		CreatedAt: time.Date(2025, 1, 1, 11, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, 1, 1, 11, 0, 0, 0, time.UTC),
	}
	ch.mu.Unlock()

	ch.sweepSessions()

	ch.mu.Lock()
	defer ch.mu.Unlock()
	if _, ok := ch.sessions["old-session"]; ok {
		t.Fatal("old-session should have been swept")
	}
	if _, ok := ch.sessions["fresh-session"]; !ok {
		t.Fatal("fresh-session should not have been swept")
	}
}

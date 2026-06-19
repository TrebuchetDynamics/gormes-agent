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
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/network/vpnhost"
)

func TestNavivoxE2ERejectsWeakPublicTokenBeforeListen(t *testing.T) {
	_, err := NewChannel(config.NavivoxCfg{
		Enabled:         true,
		GatewayID:       "gw_0123456789abcdef0123456789abcdef",
		BindHost:        "0.0.0.0",
		Port:            config.NavivoxDefaultPort,
		ExposureMode:    config.NavivoxExposurePublic,
		AuthMode:        config.NavivoxAuthPairingToken,
		Token:           "nvbx_test_token",
		PublicConfirmed: true,
		AllowOrigins:    []string{"https://navivox.example"},
	}, nil)
	if err == nil {
		t.Fatal("NewChannel error = nil, want weak public token rejection")
	}
	for _, want := range []string{"navivox.token", "public", "at least"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, missing %q", err, want)
		}
	}
}

func TestNavivoxE2ERejectsLayeredTailscaleIdentityAuthOnPublicExposure(t *testing.T) {
	_, err := NewChannel(config.NavivoxCfg{
		Enabled:         true,
		GatewayID:       "gw_0123456789abcdef0123456789abcdef",
		BindHost:        "0.0.0.0",
		Port:            config.NavivoxDefaultPort,
		ExposureMode:    config.NavivoxExposurePublic,
		AuthMode:        config.NavivoxAuthTokenAndTailscaleIdentity,
		Token:           strongNavivoxTokenForTests,
		PublicConfirmed: true,
		AllowOrigins:    []string{"https://navivox.example"},
	}, nil)
	if err == nil {
		t.Fatal("NewChannel error = nil, want public token+tailscale identity rejection")
	}
	for _, want := range []string{"token_and_tailscale_identity", "tailscale"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, missing %q", err, want)
		}
	}
}

func TestNavivoxE2ERejectsConflictingTailscaleIdentityHeaders(t *testing.T) {
	prev := vpnHostLister
	t.Cleanup(func() { vpnHostLister = prev })
	vpnHostLister = func(context.Context) ([]vpnhost.Host, error) {
		return []vpnhost.Host{{Iface: "tailscale0", Kind: vpnhost.KindTailscale, IPv4: "100.64.1.2"}}, nil
	}
	ch, err := NewChannel(config.NavivoxCfg{
		Enabled:                  true,
		GatewayID:                "gw_0123456789abcdef0123456789abcdef",
		BindHost:                 "100.64.1.2",
		Port:                     config.NavivoxDefaultPort,
		ExposureMode:             config.NavivoxExposureTailscale,
		AuthMode:                 config.NavivoxAuthTokenAndTailscaleIdentity,
		Token:                    strongNavivoxTokenForTests,
		AllowedTailnetIdentities: []string{"juan@example.com"},
		AllowOrigins:             []string{"https://navivox.example"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/v1/navivox/status", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+strongNavivoxTokenForTests)
	req.Header.Set("Tailscale-User-Login", "juan@example.com")
	req.Header.Set("X-Tailscale-User-Login", "intruder@example.com")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("conflicting tailscale identity headers status = %d, want 401", resp.StatusCode)
	}
}

func TestNavivoxE2ETokenAuthRequiresHeaderOrWebSocketProtocolNeverURLQuery(t *testing.T) {
	ch, err := NewChannel(config.NavivoxCfg{
		Enabled:      true,
		GatewayID:    "gw_0123456789abcdef0123456789abcdef",
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
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()

	queryOnly, err := http.Get(server.URL + "/v1/navivox/status?token=nvbx_test_token")
	if err != nil {
		t.Fatal(err)
	}
	defer queryOnly.Body.Close()
	if queryOnly.StatusCode != http.StatusUnauthorized {
		t.Fatalf("query token status = %d, want 401", queryOnly.StatusCode)
	}

	req, err := http.NewRequest(http.MethodGet, server.URL+"/v1/navivox/status", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "bearer nvbx_test_token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("lowercase bearer status = %d, want 200", resp.StatusCode)
	}
}

func TestNavivoxE2EAuthFailuresRateLimitRemoteHost(t *testing.T) {
	ch, err := NewChannel(config.NavivoxCfg{
		Enabled:      true,
		GatewayID:    "gw_0123456789abcdef0123456789abcdef",
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
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	ch.now = func() time.Time { return now }
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()

	for i := 0; i < navivoxAuthFailureLimit; i++ {
		req, err := http.NewRequest(http.MethodGet, server.URL+"/v1/navivox/status", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer wrong-token")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("bad token attempt %d status = %d, want 401", i+1, resp.StatusCode)
		}
	}

	validReq, err := http.NewRequest(http.MethodGet, server.URL+"/v1/navivox/status", nil)
	if err != nil {
		t.Fatal(err)
	}
	validReq.Header.Set("Authorization", "Bearer nvbx_test_token")
	limited, err := http.DefaultClient.Do(validReq)
	if err != nil {
		t.Fatal(err)
	}
	defer limited.Body.Close()
	if limited.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("valid token during auth lockout status = %d, want 429", limited.StatusCode)
	}

	now = now.Add(navivoxAuthFailureWindow + time.Second)
	validReq2, err := http.NewRequest(http.MethodGet, server.URL+"/v1/navivox/status", nil)
	if err != nil {
		t.Fatal(err)
	}
	validReq2.Header.Set("Authorization", "Bearer nvbx_test_token")
	okResp, err := http.DefaultClient.Do(validReq2)
	if err != nil {
		t.Fatal(err)
	}
	defer okResp.Body.Close()
	if okResp.StatusCode != http.StatusOK {
		t.Fatalf("valid token after auth lockout status = %d, want 200", okResp.StatusCode)
	}
}

func TestNavivoxE2EURLCredentialProbingIsRateLimited(t *testing.T) {
	ch, err := NewChannel(config.NavivoxCfg{
		Enabled:      true,
		GatewayID:    "gw_0123456789abcdef0123456789abcdef",
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
	now := time.Date(2026, 6, 4, 9, 0, 0, 0, time.UTC)
	ch.now = func() time.Time { return now }
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()

	for i := 0; i < navivoxAuthFailureLimit; i++ {
		resp, err := http.Get(server.URL + "/v1/navivox/status?token=nvbx_test_token")
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("url credential attempt %d status = %d, want 401", i+1, resp.StatusCode)
		}
	}

	locked, err := http.Get(server.URL + "/v1/navivox/status?token=nvbx_test_token")
	if err != nil {
		t.Fatal(err)
	}
	defer locked.Body.Close()
	if locked.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("url credential during auth lockout status = %d, want 429", locked.StatusCode)
	}
}

func TestNavivoxE2ERejectsURLCredentialEvenWithValidTokenAuth(t *testing.T) {
	ch, err := NewChannel(config.NavivoxCfg{
		Enabled:      true,
		GatewayID:    "gw_0123456789abcdef0123456789abcdef",
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
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/v1/navivox/status?token=nvbx_test_token", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer nvbx_test_token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status query token with valid header = %d, want 401", resp.StatusCode)
	}

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/v1/navivox/stream?token=nvbx_test_token"
	dialer := websocket.Dialer{Subprotocols: []string{
		navivoxWebSocketProtocol,
		navivoxWebSocketTokenProtocolPrefix + base64.RawURLEncoding.EncodeToString([]byte("nvbx_test_token")),
	}}
	conn, wsResp, err := dialer.Dial(wsURL, nil)
	if err == nil {
		_ = conn.Close()
		t.Fatal("websocket query token with valid subprotocol auth succeeded, want rejection")
	}
	if wsResp == nil || wsResp.StatusCode != http.StatusUnauthorized {
		status := 0
		if wsResp != nil {
			status = wsResp.StatusCode
		}
		t.Fatalf("websocket query token status = %d err=%v, want 401", status, err)
	}
}

func TestNavivoxE2EHTTPTurnRequiresJSONContentType(t *testing.T) {
	ch, err := NewChannel(config.NavivoxCfg{
		Enabled:      true,
		GatewayID:    "gw_0123456789abcdef0123456789abcdef",
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
	inbox := make(chan gateway.InboundEvent, 1)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/navivox/turn", strings.NewReader(`{"request_id":"req-json","text":"hello"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer nvbx_test_token")
	req.Header.Set("Content-Type", "text/plain")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("text/plain turn status = %d, want 415", resp.StatusCode)
	}
	select {
	case ev := <-inbox:
		t.Fatalf("text/plain turn enqueued event = %+v, want none", ev)
	default:
	}
}

func TestNavivoxE2EHTTPTurnRejectsOversizedRequestID(t *testing.T) {
	ch, err := NewChannel(config.NavivoxCfg{
		Enabled:      true,
		GatewayID:    "gw_0123456789abcdef0123456789abcdef",
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
	inbox := make(chan gateway.InboundEvent, 1)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()

	requestID := strings.Repeat("r", 129)
	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/navivox/turn", strings.NewReader(`{"request_id":"`+requestID+`","text":"hello"}`))
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
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("oversized request_id status = %d, want 400", resp.StatusCode)
	}
	select {
	case ev := <-inbox:
		t.Fatalf("oversized request_id enqueued event = %+v, want none", ev)
	default:
	}
}

func TestNavivoxE2EHTTPTurnRejectsOversizedSessionID(t *testing.T) {
	ch, err := NewChannel(config.NavivoxCfg{
		Enabled:      true,
		GatewayID:    "gw_0123456789abcdef0123456789abcdef",
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
	inbox := make(chan gateway.InboundEvent, 1)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()

	sessionID := strings.Repeat("s", 257)
	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/navivox/turn", strings.NewReader(`{"request_id":"req-session-too-long","session_id":"`+sessionID+`","text":"hello"}`))
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
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("oversized session_id status = %d, want 400", resp.StatusCode)
	}
	select {
	case ev := <-inbox:
		t.Fatalf("oversized session_id enqueued event = %+v, want none", ev)
	default:
	}
}

func TestNavivoxE2EHTTPTurnRejectsTrailingJSON(t *testing.T) {
	ch, err := NewChannel(config.NavivoxCfg{
		Enabled:      true,
		GatewayID:    "gw_0123456789abcdef0123456789abcdef",
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
	inbox := make(chan gateway.InboundEvent, 1)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/navivox/turn", strings.NewReader(`{"request_id":"req-one","text":"first"}{"request_id":"req-two","text":"second"}`))
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
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("trailing JSON turn status = %d, want 400", resp.StatusCode)
	}
	select {
	case ev := <-inbox:
		t.Fatalf("trailing JSON turn enqueued event = %+v, want none", ev)
	default:
	}
}

func TestNavivoxE2ERejectsDuplicateTokenCredentialSources(t *testing.T) {
	ch, err := NewChannel(config.NavivoxCfg{
		Enabled:      true,
		GatewayID:    "gw_0123456789abcdef0123456789abcdef",
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
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/v1/navivox/status", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer nvbx_test_token")
	req.Header.Set("X-Gormes-Navivox-Token", "nvbx_test_token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("duplicate token source status = %d, want 401", resp.StatusCode)
	}
}

func TestNavivoxE2ERejectsDuplicateAuthorizationHeaders(t *testing.T) {
	ch, err := NewChannel(config.NavivoxCfg{
		Enabled:      true,
		GatewayID:    "gw_0123456789abcdef0123456789abcdef",
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
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/v1/navivox/status", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer nvbx_test_token")
	req.Header.Add("Authorization", "Bearer nvbx_test_token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("duplicate Authorization status = %d, want 401", resp.StatusCode)
	}
}

func TestNavivoxE2ERejectsAuthenticatedWebSocketWithoutNavivoxProtocol(t *testing.T) {
	ch, err := NewChannel(config.NavivoxCfg{
		Enabled:      true,
		GatewayID:    "gw_0123456789abcdef0123456789abcdef",
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
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/v1/navivox/stream"
	dialer := websocket.Dialer{}
	conn, resp, err := dialer.Dial(wsURL, http.Header{"Authorization": []string{"Bearer nvbx_test_token"}})
	if err == nil {
		_ = conn.Close()
		t.Fatal("websocket dial without navivox.v1 subprotocol succeeded, want rejection")
	}
	if resp == nil || resp.StatusCode != http.StatusBadRequest {
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		t.Fatalf("websocket missing protocol status = %d err=%v, want 400", status, err)
	}
}

func TestNavivoxE2ERejectsOversizedHTTPTurnWithTyped413(t *testing.T) {
	ch, err := NewChannel(config.NavivoxCfg{
		Enabled:      true,
		GatewayID:    "gw_0123456789abcdef0123456789abcdef",
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
	inbox := make(chan gateway.InboundEvent, 1)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()
	httpc := newNavivoxHTTPContract(t, server.URL)

	var event ServerEvent
	body := `{"request_id":"req-http-too-big","text":"` + strings.Repeat("x", navivoxMaxTurnRequestBytes+1) + `"}`
	httpc.JSON(http.MethodPost, "/v1/navivox/turn", body, http.StatusRequestEntityTooLarge, &event)
	if event.Type != "error" || event.Code != "request_too_large" {
		t.Fatalf("oversized HTTP turn event = %+v, want request_too_large", event)
	}
	select {
	case ev := <-inbox:
		t.Fatalf("oversized HTTP turn enqueued gateway event: %+v", ev)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestNavivoxE2ERejectsOversizedWebSocketTurnBeforeGatewayEnqueue(t *testing.T) {
	ch, err := NewChannel(config.NavivoxCfg{
		Enabled:      true,
		GatewayID:    "gw_0123456789abcdef0123456789abcdef",
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
	inbox := make(chan gateway.InboundEvent, 1)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/v1/navivox/stream"
	dialer := websocket.Dialer{Subprotocols: []string{
		navivoxWebSocketProtocol,
		navivoxWebSocketTokenProtocolPrefix + base64.RawURLEncoding.EncodeToString([]byte("nvbx_test_token")),
	}}
	conn, resp, err := dialer.Dial(wsURL, nil)
	if err != nil {
		if resp != nil {
			t.Fatalf("websocket dial status=%d err=%v", resp.StatusCode, err)
		}
		t.Fatal(err)
	}
	defer conn.Close()

	payload := `{"type":"start_turn","request_id":"req-too-big","text":"` + strings.Repeat("x", navivoxMaxTurnRequestBytes+1) + `"}`
	if err := conn.WriteMessage(websocket.TextMessage, []byte(payload)); err != nil {
		t.Fatal(err)
	}
	var event ServerEvent
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatal(err)
	}
	if event.Type != "error" || event.Code != "request_too_large" || event.RequestID != "req-too-big" {
		t.Fatalf("oversized websocket event = %+v, want request_too_large error", event)
	}
	select {
	case ev := <-inbox:
		t.Fatalf("oversized websocket turn enqueued gateway event: %+v", ev)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestNavivoxE2ERejectsUntrustedBrowserOriginEvenWithValidToken(t *testing.T) {
	ch, err := NewChannel(config.NavivoxCfg{
		Enabled:      true,
		GatewayID:    "gw_0123456789abcdef0123456789abcdef",
		BindHost:     config.NavivoxDefaultBindHost,
		Port:         config.NavivoxDefaultPort,
		ExposureMode: config.NavivoxExposureLocal,
		AuthMode:     config.NavivoxAuthPairingToken,
		Token:        "nvbx_test_token",
		AllowOrigins: []string{"https://navivox.example"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/v1/navivox/status", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer nvbx_test_token")
	req.Header.Set("Origin", "https://evil.example")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("untrusted origin status = %d, want 403", resp.StatusCode)
	}

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/v1/navivox/stream"
	dialer := websocket.Dialer{Subprotocols: []string{
		navivoxWebSocketProtocol,
		navivoxWebSocketTokenProtocolPrefix + base64.RawURLEncoding.EncodeToString([]byte("nvbx_test_token")),
	}}
	_, wsResp, err := dialer.Dial(wsURL, http.Header{"Origin": []string{"https://evil.example"}})
	if err == nil {
		t.Fatal("websocket dial from untrusted origin succeeded, want rejection")
	}
	if wsResp == nil || wsResp.StatusCode != http.StatusForbidden {
		status := 0
		if wsResp != nil {
			status = wsResp.StatusCode
		}
		t.Fatalf("websocket untrusted origin status = %d err=%v, want 403", status, err)
	}
}

func TestNavivoxE2EPublicHTTPStatusAdvertisesInsecureTransport(t *testing.T) {
	ch, err := NewChannel(config.NavivoxCfg{
		Enabled:         true,
		GatewayID:       "gw_0123456789abcdef0123456789abcdef",
		BindHost:        "0.0.0.0",
		Port:            config.NavivoxDefaultPort,
		ExposureMode:    config.NavivoxExposurePublic,
		AuthMode:        config.NavivoxAuthPairingToken,
		Token:           strongNavivoxTokenForTests,
		PublicConfirmed: true,
		AllowOrigins:    []string{"https://navivox.example"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()
	httpc := newNavivoxHTTPContractWithToken(t, server.URL, strongNavivoxTokenForTests)

	var status struct {
		TransportSecurity struct {
			EffectiveSecurity         string `json:"effective_security"`
			ExposureMode              string `json:"exposure_mode"`
			DurableCredentialsAllowed bool   `json:"durable_credentials_allowed"`
		} `json:"transport_security"`
	}
	httpc.JSON(http.MethodGet, "/v1/navivox/status", "", http.StatusOK, &status)
	if status.TransportSecurity.EffectiveSecurity != "insecure" || status.TransportSecurity.ExposureMode != config.NavivoxExposurePublic || status.TransportSecurity.DurableCredentialsAllowed {
		t.Fatalf("public transport security = %+v, want insecure session-only status", status.TransportSecurity)
	}
}

func TestNavivoxE2ECapabilitiesAdvertiseDurableReconnectOnLoopback(t *testing.T) {
	ch, err := NewChannel(config.NavivoxCfg{
		Enabled:      true,
		GatewayID:    "gw_0123456789abcdef0123456789abcdef",
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
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()
	httpc := newNavivoxHTTPContract(t, server.URL)

	var caps struct {
		DurableReconnect struct {
			Supported         bool     `json:"supported"`
			IssueEndpoint     string   `json:"issue_endpoint"`
			ListEndpoint      string   `json:"list_endpoint"`
			RevokeEndpoint    string   `json:"revoke_endpoint"`
			AuthMethods       []string `json:"auth_methods"`
			Scopes            []string `json:"scopes"`
			Platforms         []string `json:"platforms"`
			Interim           bool     `json:"interim"`
			EffectiveSecurity string   `json:"effective_security"`
			BlockedReason     string   `json:"blocked_reason"`
		} `json:"durable_reconnect"`
	}
	httpc.JSON(http.MethodGet, "/v1/navivox/capabilities", "", http.StatusOK, &caps)
	if !caps.DurableReconnect.Supported {
		t.Fatalf("durable_reconnect.supported = false on loopback, want supported now issuance exists: %+v", caps.DurableReconnect)
	}
	if caps.DurableReconnect.IssueEndpoint != "/v1/navivox/device-credentials" ||
		caps.DurableReconnect.RevokeEndpoint != "/v1/navivox/device-credentials/revoke" {
		t.Fatalf("durable reconnect issue contract = issue=%q revoke=%q, want device-credentials endpoints", caps.DurableReconnect.IssueEndpoint, caps.DurableReconnect.RevokeEndpoint)
	}
	if len(caps.DurableReconnect.AuthMethods) == 0 || !caps.DurableReconnect.Interim {
		t.Fatalf("durable reconnect auth=%v interim=%v, want non-empty interim auth methods", caps.DurableReconnect.AuthMethods, caps.DurableReconnect.Interim)
	}
	if caps.DurableReconnect.EffectiveSecurity != "loopback" {
		t.Fatalf("durable_reconnect.effective_security = %q, want loopback", caps.DurableReconnect.EffectiveSecurity)
	}
	if caps.DurableReconnect.BlockedReason != "" {
		t.Fatalf("durable_reconnect.blocked_reason = %q, want empty when supported", caps.DurableReconnect.BlockedReason)
	}
}

func TestNavivoxE2EAuthenticatedClientSeesGatewayIdentityAndRunsScopedTurn(t *testing.T) {
	const gatewayID = "gw_0123456789abcdef0123456789abcdef"
	ch, err := NewChannel(config.NavivoxCfg{
		Enabled:      true,
		GatewayID:    gatewayID,
		GatewayLabel: "Kitchen Gormes",
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
	ch.loadContacts = func(context.Context) ([]ProfileContact, error) {
		return []ProfileContact{{
			ServerID:           "local-gormes",
			ProfileID:          "mineru",
			DisplayName:        "Mineru Builder",
			ServerLabel:        "Kitchen Gormes",
			LatestPreview:      "Ready for E2E",
			LatestPreviewKind:  "status",
			Health:             ProfileContactHealthOnline,
			WorkspaceRootCount: 1,
			WorkspaceRootsOK:   true,
			MicAvailable:       true,
			ActiveTurnState:    ProfileContactTurnIdle,
		}}, nil
	}
	inbox := make(chan gateway.InboundEvent, 1)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()
	httpc := newNavivoxHTTPContract(t, server.URL)

	var status struct {
		GatewayID         string `json:"gateway_id"`
		GatewayLabel      string `json:"gateway_label"`
		TransportSecurity struct {
			EffectiveSecurity         string `json:"effective_security"`
			ExposureMode              string `json:"exposure_mode"`
			TLS                       bool   `json:"tls"`
			PrivateNetwork            bool   `json:"private_network"`
			DurableCredentialsAllowed bool   `json:"durable_credentials_allowed"`
		} `json:"transport_security"`
	}
	httpc.JSON(http.MethodGet, "/v1/navivox/status", "", http.StatusOK, &status)
	if status.GatewayID != gatewayID || status.GatewayLabel != "Kitchen Gormes" {
		t.Fatalf("status identity = %+v, want authenticated gateway identity and label", status)
	}
	if status.TransportSecurity.EffectiveSecurity != "loopback" || status.TransportSecurity.ExposureMode != config.NavivoxExposureLocal || status.TransportSecurity.TLS || status.TransportSecurity.PrivateNetwork || !status.TransportSecurity.DurableCredentialsAllowed {
		t.Fatalf("transport security = %+v, want loopback durable-allowed status", status.TransportSecurity)
	}

	var contacts profileContactSnapshot
	httpc.JSON(http.MethodGet, "/v1/navivox/profile-contacts", "", http.StatusOK, &contacts)
	if len(contacts.Contacts) != 1 || contacts.Contacts[0].ServerID != "local-gormes" || contacts.Contacts[0].ServerLabel != "Kitchen Gormes" {
		t.Fatalf("profile contacts = %+v, want server-scoped contact with gateway display label", contacts.Contacts)
	}
	if contacts.Contacts[0].ServerID == status.GatewayID {
		t.Fatalf("profile contact server_id %q must stay distinct from gateway_id %q", contacts.Contacts[0].ServerID, status.GatewayID)
	}

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/v1/navivox/stream"
	dialer := websocket.Dialer{Subprotocols: []string{
		navivoxWebSocketProtocol,
		navivoxWebSocketTokenProtocolPrefix + base64.RawURLEncoding.EncodeToString([]byte("nvbx_test_token")),
	}}
	conn, resp, err := dialer.Dial(wsURL, nil)
	if err != nil {
		if resp != nil {
			t.Fatalf("websocket dial status=%d err=%v", resp.StatusCode, err)
		}
		t.Fatal(err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(ClientMessage{
		Type:      "start_turn",
		RequestID: "req-e2e",
		Text:      "hello e2e gateway",
		Metadata:  map[string]any{"server_id": "local-gormes", "profile_id": "mineru"},
	}); err != nil {
		t.Fatal(err)
	}
	started := readNavivoxE2EEvent(t, conn)
	if started.Type != "session_started" || started.RequestID != "req-e2e" || started.SessionID == "" {
		t.Fatalf("session_started = %+v", started)
	}

	select {
	case ev := <-inbox:
		if ev.Platform != PlatformName || ev.Kind != gateway.EventSubmit || ev.ChatID != started.SessionID || ev.MsgID != "req-e2e" || ev.Text != "hello e2e gateway" {
			t.Fatalf("gateway event = %+v, want scoped Navivox submit", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for gateway turn")
	}

	if _, err := ch.Send(context.Background(), started.SessionID, "hello from kitchen gateway"); err != nil {
		t.Fatal(err)
	}
	seenAssistant := false
	seenDone := false
	deadline := time.After(time.Second)
	for !(seenAssistant && seenDone) {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for assistant/done events; assistant=%v done=%v", seenAssistant, seenDone)
		default:
		}
		event := readNavivoxE2EEvent(t, conn)
		switch event.Type {
		case "profile_contact_update":
			if event.Contact == nil || event.Contact.ServerID != "local-gormes" || event.Contact.ProfileID != "mineru" {
				t.Fatalf("profile contact update = %+v", event.Contact)
			}
		case "assistant_message":
			if event.Text != "hello from kitchen gateway" || event.SessionID != started.SessionID {
				t.Fatalf("assistant event = %+v", event)
			}
			seenAssistant = true
		case "done":
			if event.SessionID != started.SessionID {
				t.Fatalf("done event = %+v", event)
			}
			seenDone = true
		}
	}
}

func readNavivoxE2EEvent(t *testing.T, conn *websocket.Conn) ServerEvent {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	var event ServerEvent
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatal(err)
	}
	return event
}

package navivox

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/network/vpnhost"
)

const strongNavivoxTokenForTests = "nvbx_0123456789abcdef0123456789abcdef"

func TestNavivoxStatusRequiresAuthAndHealthzIsPublic(t *testing.T) {
	ch := newTestChannel(t)
	inbox := make(chan gateway.InboundEvent, 1)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()
	httpc := newNavivoxHTTPContract(t, server.URL)

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

	var payload map[string]any
	httpc.JSON(http.MethodGet, "/v1/navivox/status", "", http.StatusOK, &payload)
	if payload["protocol_version"] != "navivox.v1" {
		t.Fatalf("protocol_version = %v, want navivox.v1", payload["protocol_version"])
	}
	protocols, ok := payload["websocket_protocols"].([]any)
	if !ok || len(protocols) != 1 || protocols[0] != "navivox.v1" {
		t.Fatalf("websocket_protocols = %#v, want current Navivox protocol only", payload["websocket_protocols"])
	}
	capabilities, ok := payload["capabilities"].([]any)
	if !ok || !containsAny(capabilities, "profile_contacts") || !containsAny(capabilities, "turn_control") {
		t.Fatalf("capabilities = %#v, want profile_contacts and turn_control", payload["capabilities"])
	}
}

func TestNavivoxStatusIncludesStableGatewayIdentityDistinctFromProfileServers(t *testing.T) {
	const gatewayID = "gw_0123456789abcdef0123456789abcdef"
	routing := config.NavivoxProfileRoutingReport{Servers: []config.NavivoxServerRoute{{
		ServerID: "navivox-gateway",
		Profiles: []config.NavivoxProfileRoute{{
			ProfileID:   "main",
			DisplayName: "Main Desk",
		}},
	}}}
	ch, err := NewChannel(config.NavivoxCfg{
		Enabled:      true,
		GatewayID:    gatewayID,
		GatewayLabel: "  Gormes gateway  ",
		BindHost:     config.NavivoxDefaultBindHost,
		Port:         config.NavivoxDefaultPort,
		ExposureMode: config.NavivoxExposureLocal,
		AuthMode:     config.NavivoxAuthPairingToken,
		Token:        "nvbx_test_token",
		AllowOrigins: []string{"*"},
	}, nil, WithProfileRouting(routing))
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()
	httpc := newNavivoxHTTPContract(t, server.URL)

	var payload struct {
		GatewayID      string `json:"gateway_id"`
		GatewayLabel   string `json:"gateway_label"`
		ProfileRouting struct {
			Servers []struct {
				ServerID string `json:"server_id"`
			} `json:"servers"`
		} `json:"profile_routing"`
	}
	httpc.JSON(http.MethodGet, "/v1/navivox/status", "", http.StatusOK, &payload)
	if payload.GatewayID != gatewayID {
		t.Fatalf("gateway_id = %q, want %q", payload.GatewayID, gatewayID)
	}
	if payload.GatewayLabel != "Gormes gateway" {
		t.Fatalf("gateway_label = %q, want bland display label", payload.GatewayLabel)
	}
	if len(payload.ProfileRouting.Servers) == 0 || payload.ProfileRouting.Servers[0].ServerID != "navivox-gateway" {
		t.Fatalf("profile routing servers = %+v, want profile-scoped server id", payload.ProfileRouting.Servers)
	}
	if payload.ProfileRouting.Servers[0].ServerID == payload.GatewayID {
		t.Fatalf("gateway_id must be distinct from profile contact server_id: %+v", payload)
	}
}

func TestNavivoxProfileSeedEndpointCreatesDraftAndApplyShowsContact(t *testing.T) {
	home := filepath.Join(t.TempDir(), "gormes")
	t.Setenv("GORMES_HOME", home)
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	ch := newTestChannel(t)
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()
	httpc := newNavivoxHTTPContract(t, server.URL)

	unauth, err := http.Post(server.URL+"/v1/navivox/profile-seed", "application/json", strings.NewReader(`{"seed":"work on mineru repo"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer unauth.Body.Close()
	if unauth.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized profile-seed status = %d, want 401", unauth.StatusCode)
	}

	var draftPayload struct {
		Action string `json:"action"`
		Status string `json:"status"`
		Draft  struct {
			ProfileID                string `json:"profile_id"`
			GenerationSource         string `json:"generation_source"`
			WorkspaceRootSuggestions []struct {
				RequiresConfirmation bool `json:"requires_confirmation"`
			} `json:"workspace_root_suggestions"`
		} `json:"draft"`
	}
	httpc.JSON(http.MethodPost, "/v1/navivox/profile-seed", `{"seed":"work on mineru repo"}`, http.StatusOK, &draftPayload)
	if draftPayload.Action != "profile_seed_draft" || draftPayload.Draft.ProfileID != "work-mineru-repo" || draftPayload.Draft.GenerationSource != "template" {
		t.Fatalf("draft payload = %+v, want template work-mineru-repo", draftPayload)
	}
	if len(draftPayload.Draft.WorkspaceRootSuggestions) == 0 || !draftPayload.Draft.WorkspaceRootSuggestions[0].RequiresConfirmation {
		t.Fatalf("workspace suggestions = %+v, want explicit confirmation required", draftPayload.Draft.WorkspaceRootSuggestions)
	}
	if _, err := os.Stat(filepath.Join(home, "profiles", "work-mineru-repo")); !os.IsNotExist(err) {
		t.Fatalf("dry-run should not create profile root; stat err=%v", err)
	}

	var applied struct {
		Action         string          `json:"action"`
		Applied        bool            `json:"applied"`
		ProfileID      string          `json:"profile_id"`
		WorkspaceCount int             `json:"workspace_count"`
		Contact        *ProfileContact `json:"contact"`
	}
	httpc.JSON(http.MethodPost, "/v1/navivox/profile-seed", `{"seed":"work on mineru repo","apply":true}`, http.StatusOK, &applied)
	if applied.Action != "profile_seed_applied" || !applied.Applied || applied.ProfileID != "work-mineru-repo" || applied.WorkspaceCount != 0 {
		t.Fatalf("apply payload = %+v, want applied profile with no implicit workspaces", applied)
	}
	if applied.Contact == nil || applied.Contact.ProfileID != "work-mineru-repo" {
		t.Fatalf("contact = %+v, want seeded profile contact", applied.Contact)
	}

	var snapshot profileContactSnapshot
	httpc.JSON(http.MethodGet, "/v1/navivox/profile-contacts", "", http.StatusOK, &snapshot)
	ids := profileContactIDs(snapshot.Contacts)
	if !slices.Contains(ids, "work-mineru-repo") {
		t.Fatalf("contact IDs = %v, want seeded profile", ids)
	}
}

func TestNavivoxProfileRoutingEndpointIsAuthBoundedAndSecretFree(t *testing.T) {
	routingSource := config.Config{Profiles: map[string]config.ProfileCfg{
		"mineru": {
			Enabled:    true,
			Name:       "Mineru Ops",
			Workspaces: []string{"/srv/gormes", "/srv/navivox"},
			Providers: map[string]config.ProfileProviderCfg{
				"openai-codex": {Enabled: true, Credential: "provider-secret-ref"},
			},
			Channels: map[string]config.ProfileChannelCfg{
				"navivox":  {Enabled: true},
				"telegram": {Enabled: true, Credential: "telegram-secret-ref"},
			},
		},
	}}
	ch, err := NewChannel(config.NavivoxCfg{
		Enabled:      true,
		BindHost:     config.NavivoxDefaultBindHost,
		Port:         config.NavivoxDefaultPort,
		ExposureMode: config.NavivoxExposureLocal,
		AuthMode:     config.NavivoxAuthPairingToken,
		Token:        "nvbx_test_token",
		AllowOrigins: []string{"*"},
	}, nil, WithProfileRouting(routingSource.NavivoxProfileRouting()))
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()

	unauth, err := http.Get(server.URL + "/v1/navivox/profile-routing")
	if err != nil {
		t.Fatal(err)
	}
	defer unauth.Body.Close()
	if unauth.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized profile-routing status = %d, want 401", unauth.StatusCode)
	}

	req, err := http.NewRequest(http.MethodGet, server.URL+"/v1/navivox/profile-routing", nil)
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
		t.Fatalf("profile-routing status = %d, want 200", resp.StatusCode)
	}
	var got config.NavivoxProfileRoutingReport
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	want := config.NavivoxProfileRoutingReport{Profiles: []config.NavivoxProfileRoute{{
		ProfileID:   "mineru",
		DisplayName: "Mineru Ops",
		Workspaces:  []string{"/srv/gormes", "/srv/navivox"},
		Providers:   []string{"openai-codex"},
		Channels:    []string{"navivox", "telegram"},
	}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("profile-routing payload = %#v, want %#v", got, want)
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"provider-secret-ref", "telegram-secret-ref", "nvbx_test_token"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("profile-routing leaked %q: %s", forbidden, raw)
		}
	}
}

func TestNavivoxStatusIncludesServerScopedProfileRoutingWithoutDefaultProfile(t *testing.T) {
	routing := config.NavivoxProfileRoutingReport{Servers: []config.NavivoxServerRoute{{
		ServerID:     "local",
		Bind:         "127.0.0.1:8787",
		Transports:   []string{"http", "ws"},
		Capabilities: []string{"connect_and_talk"},
		Profiles: []config.NavivoxProfileRoute{{
			ProfileID:              "main",
			DisplayName:            "Main Desk",
			Ready:                  true,
			CredentialConfigured:   true,
			VoiceProfileConfigured: true,
			ServerIDs:              []string{"local"},
			Channels:               []string{"navivox"},
		}},
	}}}
	ch, err := NewChannel(config.NavivoxCfg{
		Enabled:      true,
		BindHost:     config.NavivoxDefaultBindHost,
		Port:         config.NavivoxDefaultPort,
		ExposureMode: config.NavivoxExposureLocal,
		AuthMode:     config.NavivoxAuthPairingToken,
		Token:        "nvbx_test_token",
		AllowOrigins: []string{"*"},
	}, nil, WithProfileRouting(routing))
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
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"profile_routing", "server_id", "local", "profile_id", "main", "display_name", "Main Desk", "credential_configured", "voice_profile_configured"} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("status payload missing %q:\n%s", want, raw)
		}
	}
	if strings.Contains(string(raw), "default_profile") || strings.Contains(string(raw), "nvbx_test_token") {
		t.Fatalf("status payload leaked token/default profile wording:\n%s", raw)
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
	if !ok || !containsAny(capabilities, "profile_contacts") || containsAny(capabilities, "setup_handoff") {
		t.Fatalf("capabilities = %#v, want feature summary without setup_handoff", payload["capabilities"])
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
	if handoff["bridge_keepalive_required"] != true {
		t.Fatalf("bridge_keepalive_required = %#v, want true", handoff["bridge_keepalive_required"])
	}
	if handoff["bridge_lifecycle"] != "termux_pair_command" {
		t.Fatalf("bridge_lifecycle = %#v, want termux_pair_command", handoff["bridge_lifecycle"])
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
		Token:                    strongNavivoxTokenForTests,
		AllowedTailnetIdentities: []string{"juan@example.com"},
		AllowOrigins:             []string{"https://navivox.example"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	inbox := make(chan gateway.InboundEvent, 1)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()

	for name, headers := range map[string]map[string]string{
		"token-only": {
			"Authorization": "Bearer " + strongNavivoxTokenForTests,
		},
		"identity-only": {
			"Tailscale-User-Login": "juan@example.com",
		},
		"wrong-identity": {
			"Authorization":        "Bearer " + strongNavivoxTokenForTests,
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
	req.Header.Set("Authorization", "Bearer "+strongNavivoxTokenForTests)
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

func TestNavivoxRunRecordEndpointReturnsRedactedVoiceAndToolTimeline(t *testing.T) {
	ch := newTestChannel(t)
	inbox := make(chan gateway.InboundEvent, 1)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()

	reqBody := `{
		"request_id":"req-run-record",
		"session_id":"s-run-record",
		"text":"transcribed voice command",
		"metadata":{
			"input_kind":"voice",
			"audio_duration_ms":1200,
			"audio_codec":"audio/opus",
			"stt_provider":"device",
			"tts_provider":"local",
			"raw_audio_bytes":"raw-audio-bytes-must-not-leak",
			"provider_api_key":"secret-token-must-not-leak"
		}
	}`
	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/navivox/turn", strings.NewReader(reqBody))
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
	case <-inbox:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for queued gateway event")
	}

	if _, err := ch.SendToolProgress(context.Background(), "s-run-record", gateway.ToolProgressEvent{
		ID:       "tool-1",
		ToolName: "read_file",
		Status:   gateway.ToolProgressFinished,
		Summary:  "Read README",
		Metadata: map[string]any{"artifact_ref": "artifact://readme", "secret_token": "must-not-leak"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := ch.Send(context.Background(), "s-run-record", "assistant final answer"); err != nil {
		t.Fatal(err)
	}

	getReq, err := http.NewRequest(http.MethodGet, server.URL+"/v1/navivox/run-records/req-run-record", nil)
	if err != nil {
		t.Fatal(err)
	}
	getReq.Header.Set("Authorization", "Bearer nvbx_test_token")
	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		t.Fatal(err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("run-record status = %d, want 200", getResp.StatusCode)
	}
	var got struct {
		Record struct {
			RunID  string `json:"run_id"`
			Status string `json:"status"`
			Voice  struct {
				DeviceTranscript string `json:"device_transcript"`
				Audio            struct {
					DurationMS     int    `json:"duration_ms"`
					RawAudioStored bool   `json:"raw_audio_stored"`
					Retention      string `json:"retention"`
				} `json:"audio"`
			} `json:"voice"`
			ProviderUsage struct {
				Status string `json:"status"`
			} `json:"provider_usage"`
			ProviderCost struct {
				Status string `json:"status"`
			} `json:"provider_cost"`
			Transcript []struct {
				Role string `json:"role"`
				Text string `json:"text"`
			} `json:"transcript"`
			ToolEvents []struct {
				ToolCallID string         `json:"tool_call_id"`
				Name       string         `json:"name"`
				Status     string         `json:"status"`
				Metadata   map[string]any `json:"metadata"`
			} `json:"tool_events"`
		} `json:"run_record"`
	}
	if err := json.NewDecoder(getResp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Record.RunID != "req-run-record" || got.Record.Status != "completed" {
		t.Fatalf("run record identity/status = %+v", got.Record)
	}
	if got.Record.Voice.DeviceTranscript != "transcribed voice command" || got.Record.Voice.Audio.DurationMS != 1200 || got.Record.Voice.Audio.RawAudioStored || got.Record.Voice.Audio.Retention != "not_stored" {
		t.Fatalf("voice evidence = %+v", got.Record.Voice)
	}
	if got.Record.ProviderUsage.Status != "unknown" || got.Record.ProviderCost.Status != "unknown" {
		t.Fatalf("usage/cost = %+v/%+v, want unknown", got.Record.ProviderUsage, got.Record.ProviderCost)
	}
	if len(got.Record.Transcript) != 2 || got.Record.Transcript[0].Role != "user" || got.Record.Transcript[1].Role != "assistant" {
		t.Fatalf("transcript = %+v", got.Record.Transcript)
	}
	if len(got.Record.ToolEvents) != 1 || got.Record.ToolEvents[0].ToolCallID != "tool-1" || got.Record.ToolEvents[0].Status != "finished" {
		t.Fatalf("tool events = %+v", got.Record.ToolEvents)
	}
	if got.Record.ToolEvents[0].Metadata["artifact_ref"] != "artifact://readme" {
		t.Fatalf("tool metadata = %+v", got.Record.ToolEvents[0].Metadata)
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"raw-audio-bytes-must-not-leak", "secret-token-must-not-leak", "must-not-leak"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("run record leaked %q: %s", forbidden, raw)
		}
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
	dialer := websocket.Dialer{Subprotocols: []string{navivoxWebSocketProtocol}}
	conn, resp, err := dialer.Dial(wsURL, header)
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
	deadline := time.Now().Add(time.Second)
	defer func() { _ = conn.SetReadDeadline(time.Time{}) }()
	for {
		if err := conn.SetReadDeadline(deadline); err != nil {
			t.Fatal(err)
		}
		var event ServerEvent
		if err := conn.ReadJSON(&event); err != nil {
			t.Fatal(err)
		}
		if event.Type != "profile_contact_update" {
			return event
		}
		if time.Now().After(deadline) {
			t.Fatal("only received profile_contact_update events")
		}
	}
}

func waitForSessionSubscribers(t *testing.T, ch *Channel, sessionID string, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		ch.mu.Lock()
		state := ch.sessions[sessionID]
		got := -1
		if state != nil {
			got = state.Subscribers
		}
		ch.mu.Unlock()
		if state != nil && got == want {
			return
		}
		if time.Now().After(deadline) {
			if state == nil {
				t.Fatalf("session %q not found", sessionID)
			}
			t.Fatalf("subscribers for %s = %d, want %d", sessionID, got, want)
		}
		time.Sleep(10 * time.Millisecond)
	}
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
		Token:        strongNavivoxTokenForTests,
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
		Token:        strongNavivoxTokenForTests,
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
		ProfileID:        "main",
		DisplayName:      "Gormes profile",
		Health:           ProfileContactHealthWarning,
		AttentionBadges:  []string{"config", "workspace"},
		WorkspaceRootsOK: false,
	}
	overlay := ProfileContact{
		ServerID:        "navivox-gateway",
		ProfileID:       "main",
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

func TestNavivoxLocalCORSAllowsLoopbackBrowserOriginWithoutExplicitAllowlist(t *testing.T) {
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
	inbox := make(chan gateway.InboundEvent, 1)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()

	origin := "http://127.0.0.1:8767"
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
}

func TestNavivoxLocalCORSRejectsNonLoopbackBrowserOriginWithoutExplicitAllowlist(t *testing.T) {
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
	inbox := make(chan gateway.InboundEvent, 1)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()

	preflight, err := http.NewRequest(http.MethodOptions, server.URL+"/v1/navivox/status", nil)
	if err != nil {
		t.Fatal(err)
	}
	preflight.Header.Set("Origin", "https://navivox.example")
	preflightResp, err := http.DefaultClient.Do(preflight)
	if err != nil {
		t.Fatal(err)
	}
	defer preflightResp.Body.Close()
	if preflightResp.StatusCode != http.StatusForbidden {
		t.Fatalf("CORS preflight status = %d, want 403", preflightResp.StatusCode)
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
	if actualResp.StatusCode != http.StatusOK {
		t.Fatalf("CORS actual status = %d, want 200", actualResp.StatusCode)
	}
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

	waitForSessionSubscribers(t, ch, "s-sub", 1)

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

	waitForSessionSubscribers(t, ch, "s-sub", 1)

	conn1.Close()
	waitForSessionSubscribers(t, ch, "s-sub", 0)
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

func TestNavivoxTurnInProgressRejectedForDifferentDevice(t *testing.T) {
	ch := newTestChannel(t)
	inbox := make(chan gateway.InboundEvent, 4)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()
	httpc := newNavivoxHTTPContract(t, server.URL)

	var credA, credB struct {
		CredentialID string `json:"credential_id"`
		Secret       string `json:"secret"`
	}
	httpc.JSON(http.MethodPost, "/v1/navivox/device-credentials", `{"app_install_id":"device-a"}`, http.StatusCreated, &credA)
	httpc.JSON(http.MethodPost, "/v1/navivox/device-credentials", `{"app_install_id":"device-b"}`, http.StatusCreated, &credB)

	dialWithBearer := func(t *testing.T, credID, secret string) *websocket.Conn {
		t.Helper()
		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/v1/navivox/stream"
		header := http.Header{"Authorization": {"Bearer " + credID + ":" + secret}}
		dialer := websocket.Dialer{Subprotocols: []string{navivoxWebSocketProtocol}}
		conn, _, err := dialer.Dial(wsURL, header)
		if err != nil {
			t.Fatalf("dial as %s: %v", credID, err)
		}
		return conn
	}

	connA := dialWithBearer(t, credA.CredentialID, credA.Secret)
	defer connA.Close()
	connB := dialWithBearer(t, credB.CredentialID, credB.Secret)
	defer connB.Close()

	// Device A starts a turn.
	if err := connA.WriteJSON(ClientMessage{Type: "start_turn", RequestID: "req-a", SessionID: "s-a", Text: "hello from A"}); err != nil {
		t.Fatal(err)
	}
	startedA := readNonProfileContactEvent(t, connA)
	if startedA.Type != "session_started" {
		t.Fatalf("device A start_turn = %+v, want session_started", startedA)
	}
	select {
	case <-inbox:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for device A's turn in inbox")
	}

	// Device B tries to start a turn while Device A holds the active turn.
	if err := connB.WriteJSON(ClientMessage{Type: "start_turn", RequestID: "req-b", SessionID: "s-b", Text: "hello from B"}); err != nil {
		t.Fatal(err)
	}
	rejected := readNonProfileContactEvent(t, connB)
	if rejected.Type != "error" || rejected.Code != "turn_in_progress" {
		t.Fatalf("device B got %+v, want error turn_in_progress", rejected)
	}

	// Device A's turn completes via done broadcast.
	msgID, err := ch.SendPlaceholder(context.Background(), "s-a")
	if err != nil {
		t.Fatal(err)
	}
	if err := ch.EditMessageFinal(context.Background(), "s-a", msgID, "response", true); err != nil {
		t.Fatal(err)
	}
	for {
		var ev ServerEvent
		if err := connA.ReadJSON(&ev); err != nil {
			t.Fatal(err)
		}
		if ev.Type == "done" {
			break
		}
	}

	// Device B can now start a turn once Device A's turn is done.
	if err := connB.WriteJSON(ClientMessage{Type: "start_turn", RequestID: "req-b2", SessionID: "s-b2", Text: "hello from B again"}); err != nil {
		t.Fatal(err)
	}
	startedB := readNonProfileContactEvent(t, connB)
	if startedB.Type != "session_started" {
		t.Fatalf("device B second attempt = %+v, want session_started after A's turn ended", startedB)
	}
}

func TestNavivoxTurnInProgressRejectedForSameDevice(t *testing.T) {
	ch := newTestChannel(t)
	inbox := make(chan gateway.InboundEvent, 4)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()
	httpc := newNavivoxHTTPContract(t, server.URL)

	var cred struct {
		CredentialID string `json:"credential_id"`
		Secret       string `json:"secret"`
	}
	httpc.JSON(http.MethodPost, "/v1/navivox/device-credentials", `{"app_install_id":"device-a"}`, http.StatusCreated, &cred)

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/v1/navivox/stream"
	header := http.Header{"Authorization": {"Bearer " + cred.CredentialID + ":" + cred.Secret}}
	dialer := websocket.Dialer{Subprotocols: []string{navivoxWebSocketProtocol}}
	conn, _, err := dialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(ClientMessage{Type: "start_turn", RequestID: "req-1", SessionID: "s-1", Text: "first turn"}); err != nil {
		t.Fatal(err)
	}
	started := readNonProfileContactEvent(t, conn)
	if started.Type != "session_started" {
		t.Fatalf("first turn = %+v, want session_started", started)
	}
	select {
	case <-inbox:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first turn in inbox")
	}

	// Same device, different session: must be rejected.
	if err := conn.WriteJSON(ClientMessage{Type: "start_turn", RequestID: "req-2", SessionID: "s-2", Text: "second turn"}); err != nil {
		t.Fatal(err)
	}
	rejected := readNonProfileContactEvent(t, conn)
	if rejected.Type != "error" || rejected.Code != "turn_in_progress" {
		t.Fatalf("concurrent same-device turn got %+v, want error turn_in_progress", rejected)
	}
}

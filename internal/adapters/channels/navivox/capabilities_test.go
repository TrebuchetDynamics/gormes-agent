package navivox

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/internal/adaptertest"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestNavivoxCapabilitiesEndpointAdvertisesStableContract(t *testing.T) {
	ch := newTestChannel(t)
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()
	httpc := newNavivoxHTTPContract(t, server.URL)

	unauth, err := http.Get(server.URL + "/v1/navivox/capabilities")
	if err != nil {
		t.Fatal(err)
	}
	defer unauth.Body.Close()
	if unauth.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized capabilities status = %d, want 401", unauth.StatusCode)
	}

	var got struct {
		Object          string   `json:"object"`
		ProtocolVersion string   `json:"protocol_version"`
		Capabilities    []string `json:"capabilities"`
		Auth            struct {
			Mode               string   `json:"mode"`
			Headers            []string `json:"headers"`
			WebSocketProtocols []string `json:"websocket_protocols"`
		} `json:"auth"`
		Health struct {
			Aliases []string `json:"aliases"`
		} `json:"health"`
		Endpoints []struct {
			Method      string `json:"method"`
			Path        string `json:"path"`
			Auth        string `json:"auth"`
			Stability   string `json:"stability"`
			Description string `json:"description"`
		} `json:"endpoints"`
		ProfileManagement struct {
			ContactsEndpoint       string   `json:"contacts_endpoint"`
			RoutingEndpoint        string   `json:"routing_endpoint"`
			CreateFromSeedEndpoint string   `json:"create_from_seed_endpoint"`
			DashboardAPIExposed    bool     `json:"dashboard_api_exposed"`
			SupportedActions       []string `json:"supported_actions"`
			UnsupportedActions     []string `json:"unsupported_actions"`
			ProfileContractParts   []string `json:"profile_contract_parts"`
		} `json:"profile_management"`
		Attachments struct {
			MaxRequestBytes       int      `json:"max_request_bytes"`
			OpaqueUploadIDs       bool     `json:"opaque_upload_ids"`
			RawLocalPathsAccepted bool     `json:"raw_local_paths_accepted"`
			WorkspaceFileAttach   bool     `json:"workspace_file_attach"`
			MIMEAllowlist         []string `json:"mime_allowlist"`
		} `json:"attachments"`
		Voice struct {
			DeviceTranscribedTextTurns bool     `json:"device_transcribed_text_turns"`
			RawAudioUpload             bool     `json:"raw_audio_upload"`
			VoiceProfilesEndpoint      string   `json:"voice_profiles_endpoint"`
			STTProviders               []string `json:"stt_providers"`
			TTSProviders               []string `json:"tts_providers"`
		} `json:"voice"`
		Streams struct {
			CanonicalEndpoint string   `json:"canonical_endpoint"`
			EventKinds        []string `json:"event_kinds"`
			OpenAIRunsBridge  *bool    `json:"openai_runs_bridge"`
			RunsBridge        *string  `json:"runs_bridge"`
		} `json:"streams"`
	}
	httpc.JSON(http.MethodGet, "/v1/navivox/capabilities", "", http.StatusOK, &got)

	if got.Object != "gormes.navivox.capabilities" || got.ProtocolVersion != navivoxWebSocketProtocol {
		t.Fatalf("capability identity = object %q protocol %q", got.Object, got.ProtocolVersion)
	}
	if got.Auth.Mode != "pairing_token" || !adaptertest.ContainsString(got.Auth.Headers, "Authorization: Bearer <token>") {
		t.Fatalf("auth = %+v, want pairing token bearer contract", got.Auth)
	}
	if !adaptertest.ContainsString(got.Capabilities, "profile_contacts") {
		t.Fatalf("capabilities = %v, want API feature areas", got.Capabilities)
	}
	for _, forbidden := range []string{"setup_handoff", "capability_document"} {
		if adaptertest.ContainsString(got.Capabilities, forbidden) {
			t.Fatalf("capabilities = %v, %s is not a feature affordance", got.Capabilities, forbidden)
		}
	}
	removedProtocol := "gormes." + navivoxWebSocketProtocol
	if !adaptertest.ContainsString(got.Auth.WebSocketProtocols, navivoxWebSocketProtocol) || adaptertest.ContainsString(got.Auth.WebSocketProtocols, removedProtocol) {
		t.Fatalf("websocket protocols = %v, want current protocol without removed fallback", got.Auth.WebSocketProtocols)
	}
	if !adaptertest.ContainsString(got.Health.Aliases, "/healthz") || !containsEndpoint(got.Endpoints, http.MethodGet, "/v1/navivox/status") || !containsEndpoint(got.Endpoints, http.MethodGet, "/v1/navivox/capabilities") || !containsEndpoint(got.Endpoints, "WS", "/v1/navivox/stream") {
		t.Fatalf("endpoints/health missing stable status/capabilities/stream contract: health=%+v endpoints=%+v", got.Health, got.Endpoints)
	}
	for _, event := range []string{"session_started", "assistant_delta", "assistant_message", "tool_call_started", "tool_call_updated", "tool_call_finished", "profile_contact_update", "error", "done"} {
		if !adaptertest.ContainsString(got.Streams.EventKinds, event) {
			t.Fatalf("event %q missing from stream event kinds=%v", event, got.Streams.EventKinds)
		}
	}
	if got.ProfileManagement.ContactsEndpoint != "/v1/navivox/profile-contacts" || got.ProfileManagement.RoutingEndpoint != "/v1/navivox/profile-routing" || got.ProfileManagement.CreateFromSeedEndpoint != "/v1/navivox/profile-seed" {
		t.Fatalf("profile management endpoints = %+v", got.ProfileManagement)
	}
	if got.ProfileManagement.DashboardAPIExposed || !adaptertest.ContainsString(got.ProfileManagement.SupportedActions, "contact_snapshot") || !adaptertest.ContainsString(got.ProfileManagement.SupportedActions, "create_from_seed") || !adaptertest.ContainsString(got.ProfileManagement.UnsupportedActions, "direct_dashboard_api_profiles") {
		t.Fatalf("profile management support = %+v, want stable wrapper not raw dashboard profile API", got.ProfileManagement)
	}
	for _, part := range []string{"profile_contacts", "profile_routing", "voice_profiles"} {
		if !adaptertest.ContainsString(got.ProfileManagement.ProfileContractParts, part) {
			t.Fatalf("profile contract parts = %v, want %q", got.ProfileManagement.ProfileContractParts, part)
		}
	}
	if got.Attachments.MaxRequestBytes != 1<<20 || got.Attachments.RawLocalPathsAccepted || got.Attachments.WorkspaceFileAttach || got.Attachments.OpaqueUploadIDs || len(got.Attachments.MIMEAllowlist) != 0 {
		t.Fatalf("attachments = %+v, want explicit current no-upload contract", got.Attachments)
	}
	if !got.Voice.DeviceTranscribedTextTurns || got.Voice.RawAudioUpload || got.Voice.VoiceProfilesEndpoint != "/v1/navivox/voice-profiles" || len(got.Voice.TTSProviders) == 0 {
		t.Fatalf("voice = %+v, want truthful text-turn + voice-profile capabilities", got.Voice)
	}
	if got.Streams.CanonicalEndpoint != "/v1/navivox/stream" {
		t.Fatalf("canonical stream endpoint = %q", got.Streams.CanonicalEndpoint)
	}
	if got.Streams.OpenAIRunsBridge == nil || *got.Streams.OpenAIRunsBridge || got.Streams.RunsBridge != nil {
		t.Fatalf("streams bridge = openai %v legacy %v, want explicit false OpenAI runs bridge only", got.Streams.OpenAIRunsBridge, got.Streams.RunsBridge)
	}
}

func TestNavivoxCapabilitiesAuthContractFollowsActiveMode(t *testing.T) {
	tokenHeaders := []string{"Authorization: Bearer <token>", "X-Gormes-Navivox-Token"}
	tailscaleHeaders := []string{"Tailscale-User-Login", "X-Tailscale-User-Login", "Tailscale-Device-Name", "X-Tailscale-Device-Name"}
	cases := []struct {
		mode              string
		wantHeaders       []string
		forbiddenHeaders  []string
		wantTokenProtocol bool
	}{
		{
			mode:              config.NavivoxAuthPairingToken,
			wantHeaders:       tokenHeaders,
			forbiddenHeaders:  tailscaleHeaders,
			wantTokenProtocol: true,
		},
		{
			mode:              config.NavivoxAuthStaticToken,
			wantHeaders:       tokenHeaders,
			forbiddenHeaders:  tailscaleHeaders,
			wantTokenProtocol: true,
		},
		{
			mode:              config.NavivoxAuthTailscaleIdentity,
			wantHeaders:       tailscaleHeaders,
			forbiddenHeaders:  tokenHeaders,
			wantTokenProtocol: false,
		},
		{
			mode:              config.NavivoxAuthTokenAndTailscaleIdentity,
			wantHeaders:       append(append([]string{}, tokenHeaders...), tailscaleHeaders...),
			wantTokenProtocol: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.mode, func(t *testing.T) {
			ch, err := NewChannel(config.NavivoxCfg{
				Enabled:      true,
				BindHost:     config.NavivoxDefaultBindHost,
				Port:         config.NavivoxDefaultPort,
				ExposureMode: config.NavivoxExposureLocal,
				AuthMode:     tc.mode,
				Token:        "nvbx_test_token",
				AllowOrigins: []string{"*"},
			}, nil)
			if err != nil {
				t.Fatal(err)
			}
			doc := ch.capabilityDocument()
			for _, want := range tc.wantHeaders {
				if !adaptertest.ContainsString(doc.Auth.Headers, want) {
					t.Fatalf("auth headers for %s = %v, want %q", tc.mode, doc.Auth.Headers, want)
				}
			}
			for _, forbidden := range tc.forbiddenHeaders {
				if adaptertest.ContainsString(doc.Auth.Headers, forbidden) {
					t.Fatalf("auth headers for %s = %v, must not include inactive credential %q", tc.mode, doc.Auth.Headers, forbidden)
				}
			}
			tokenProtocol := navivoxWebSocketTokenProtocolPrefix + "<base64url-token>"
			if got := adaptertest.ContainsString(doc.Auth.WebSocketProtocols, tokenProtocol); got != tc.wantTokenProtocol {
				t.Fatalf("token websocket protocol for %s = %v, want %v in %v", tc.mode, got, tc.wantTokenProtocol, doc.Auth.WebSocketProtocols)
			}
		})
	}
}

func TestNavivoxStatusLinksCapabilitiesDocument(t *testing.T) {
	ch := newTestChannel(t)
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()
	httpc := newNavivoxHTTPContract(t, server.URL)

	var payload struct {
		CapabilitiesURL string   `json:"capabilities_url"`
		Capabilities    []string `json:"capabilities"`
	}
	httpc.JSON(http.MethodGet, "/v1/navivox/status", "", http.StatusOK, &payload)
	if payload.CapabilitiesURL != "/v1/navivox/capabilities" || !adaptertest.ContainsString(payload.Capabilities, "profile_contacts") || adaptertest.ContainsString(payload.Capabilities, "capability_document") || adaptertest.ContainsString(payload.Capabilities, "setup_handoff") {
		t.Fatalf("status capabilities = url %q list %v, want capability link plus feature summary", payload.CapabilitiesURL, payload.Capabilities)
	}
}

func TestNavivoxCapabilitiesRedactsSecretsAndLocalPaths(t *testing.T) {
	ch := newTestChannel(t)
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()
	httpc := newNavivoxHTTPContract(t, server.URL)

	var payload map[string]any
	httpc.JSON(http.MethodGet, "/v1/navivox/capabilities", "", http.StatusOK, &payload)
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	removedAttachmentEndpoint := "/v1/navivox/" + "uploads"
	removedAttachmentField := "unsupported" + "_until_endpoint"
	removedNotesField := "compatibility" + "_notes"
	removedAuthModesField := "accepted" + "_modes"
	removedBridgeNote := "navivox_stream_is_canonical_for_navivox_clients"
	removedTopLevelEventsField := "\"events\""
	for _, forbidden := range []string{"nvbx_test_token", "GORMES_NAVIVOX_TOKEN=", "/home/", "\\\\Users\\\\", "api_key", "password", "secret", "payload_safety", removedAttachmentField, removedAttachmentEndpoint, removedNotesField, removedAuthModesField, removedBridgeNote, removedTopLevelEventsField, "setup_handoff", "capability_document", "gormes." + navivoxWebSocketProtocol} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("capabilities leaked forbidden %q:\n%s", forbidden, raw)
		}
	}
}

func containsEndpoint(endpoints []struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Auth        string `json:"auth"`
	Stability   string `json:"stability"`
	Description string `json:"description"`
}, method, path string) bool {
	for _, endpoint := range endpoints {
		if endpoint.Method == method && endpoint.Path == path {
			return true
		}
	}
	return false
}

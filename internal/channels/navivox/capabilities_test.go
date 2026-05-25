package navivox

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestNavivoxCapabilitiesEndpointAdvertisesStableContract(t *testing.T) {
	ch := newTestChannel(t)
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()

	unauth, err := http.Get(server.URL + "/v1/navivox/capabilities")
	if err != nil {
		t.Fatal(err)
	}
	defer unauth.Body.Close()
	if unauth.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized capabilities status = %d, want 401", unauth.StatusCode)
	}

	req, err := http.NewRequest(http.MethodGet, server.URL+"/v1/navivox/capabilities", nil)
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
		t.Fatalf("capabilities status = %d, want 200", resp.StatusCode)
	}

	var got struct {
		Object          string `json:"object"`
		ProtocolVersion string `json:"protocol_version"`
		Auth            struct {
			Mode    string   `json:"mode"`
			Headers []string `json:"headers"`
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
		Events            []string `json:"events"`
		ProfileManagement struct {
			ContactsEndpoint       string   `json:"contacts_endpoint"`
			RoutingEndpoint        string   `json:"routing_endpoint"`
			CreateFromSeedEndpoint string   `json:"create_from_seed_endpoint"`
			DashboardAPIExposed    bool     `json:"dashboard_api_exposed"`
			SupportedActions       []string `json:"supported_actions"`
			UnsupportedActions     []string `json:"unsupported_actions"`
		} `json:"profile_management"`
		Attachments struct {
			MaxRequestBytes          int      `json:"max_request_bytes"`
			OpaqueUploadIDs          bool     `json:"opaque_upload_ids"`
			RawLocalPathsAccepted    bool     `json:"raw_local_paths_accepted"`
			WorkspaceFileAttach      bool     `json:"workspace_file_attach"`
			MIMEAllowlist            []string `json:"mime_allowlist"`
			UnsupportedUntilEndpoint string   `json:"unsupported_until_endpoint"`
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
		} `json:"streams"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode capabilities: %v", err)
	}

	if got.Object != "gormes.navivox.capabilities" || got.ProtocolVersion != navivoxWebSocketProtocol {
		t.Fatalf("capability identity = object %q protocol %q", got.Object, got.ProtocolVersion)
	}
	if got.Auth.Mode != "pairing_token" || !containsNavivoxCapabilityString(got.Auth.Headers, "Authorization: Bearer <token>") {
		t.Fatalf("auth = %+v, want pairing token bearer contract", got.Auth)
	}
	if !containsNavivoxCapabilityString(got.Health.Aliases, "/healthz") || !containsEndpoint(got.Endpoints, http.MethodGet, "/v1/navivox/status") || !containsEndpoint(got.Endpoints, http.MethodGet, "/v1/navivox/capabilities") || !containsEndpoint(got.Endpoints, "WS", "/v1/navivox/stream") {
		t.Fatalf("endpoints/health missing stable status/capabilities/stream contract: health=%+v endpoints=%+v", got.Health, got.Endpoints)
	}
	for _, event := range []string{"session_started", "assistant_delta", "assistant_message", "tool_call_started", "tool_call_updated", "tool_call_finished", "profile_contact_update", "error", "done"} {
		if !containsNavivoxCapabilityString(got.Events, event) || !containsNavivoxCapabilityString(got.Streams.EventKinds, event) {
			t.Fatalf("event %q missing from capabilities events=%v streams=%v", event, got.Events, got.Streams.EventKinds)
		}
	}
	if got.ProfileManagement.ContactsEndpoint != "/v1/navivox/profile-contacts" || got.ProfileManagement.RoutingEndpoint != "/v1/navivox/profile-routing" || got.ProfileManagement.CreateFromSeedEndpoint != "/v1/navivox/profile-seed" {
		t.Fatalf("profile management endpoints = %+v", got.ProfileManagement)
	}
	if got.ProfileManagement.DashboardAPIExposed || !containsNavivoxCapabilityString(got.ProfileManagement.SupportedActions, "contact_snapshot") || !containsNavivoxCapabilityString(got.ProfileManagement.SupportedActions, "create_from_seed") || !containsNavivoxCapabilityString(got.ProfileManagement.UnsupportedActions, "direct_dashboard_api_profiles") {
		t.Fatalf("profile management support = %+v, want stable wrapper not raw dashboard profile API", got.ProfileManagement)
	}
	if got.Attachments.MaxRequestBytes != 1<<20 || got.Attachments.RawLocalPathsAccepted || got.Attachments.WorkspaceFileAttach || got.Attachments.OpaqueUploadIDs || got.Attachments.UnsupportedUntilEndpoint == "" || len(got.Attachments.MIMEAllowlist) != 0 {
		t.Fatalf("attachments = %+v, want explicit unavailable durable-upload contract", got.Attachments)
	}
	if !got.Voice.DeviceTranscribedTextTurns || got.Voice.RawAudioUpload || got.Voice.VoiceProfilesEndpoint != "/v1/navivox/voice-profiles" || len(got.Voice.TTSProviders) == 0 {
		t.Fatalf("voice = %+v, want truthful text-turn + voice-profile capabilities", got.Voice)
	}
	if got.Streams.CanonicalEndpoint != "/v1/navivox/stream" {
		t.Fatalf("canonical stream endpoint = %q", got.Streams.CanonicalEndpoint)
	}
}

func TestNavivoxStatusLinksCapabilitiesDocument(t *testing.T) {
	ch := newTestChannel(t)
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
	var payload struct {
		CapabilitiesURL string   `json:"capabilities_url"`
		Capabilities    []string `json:"capabilities"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.CapabilitiesURL != "/v1/navivox/capabilities" || !containsNavivoxCapabilityString(payload.Capabilities, "capability_document") {
		t.Fatalf("status capabilities = url %q list %v, want linked capability document", payload.CapabilitiesURL, payload.Capabilities)
	}
}

func TestNavivoxCapabilitiesRedactsSecretsAndLocalPaths(t *testing.T) {
	ch := newTestChannel(t)
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/v1/navivox/capabilities", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer nvbx_test_token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"nvbx_test_token", "GORMES_NAVIVOX_TOKEN=", "/home/", "\\\\Users\\\\", "api_key", "password", "secret"} {
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

func containsNavivoxCapabilityString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

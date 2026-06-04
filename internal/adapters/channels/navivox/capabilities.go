package navivox

import (
	"net/http"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

const navivoxMaxTurnRequestBytes = 1 << 20

type capabilityEndpoint struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Auth        string `json:"auth"`
	Stability   string `json:"stability"`
	Description string `json:"description"`
}

type capabilityAuth struct {
	Mode               string   `json:"mode"`
	Headers            []string `json:"headers"`
	WebSocketProtocols []string `json:"websocket_protocols"`
}

type capabilityHealth struct {
	Canonical string   `json:"canonical"`
	Aliases   []string `json:"aliases"`
	Auth      string   `json:"auth"`
}

type capabilityProfileManagement struct {
	ContactsEndpoint       string   `json:"contacts_endpoint"`
	RoutingEndpoint        string   `json:"routing_endpoint"`
	CreateFromSeedEndpoint string   `json:"create_from_seed_endpoint"`
	DashboardAPIExposed    bool     `json:"dashboard_api_exposed"`
	SupportedActions       []string `json:"supported_actions"`
	UnsupportedActions     []string `json:"unsupported_actions"`
	ProfileContractParts   []string `json:"profile_contract_parts"`
}

type capabilityAttachments struct {
	MaxRequestBytes       int      `json:"max_request_bytes"`
	OpaqueUploadIDs       bool     `json:"opaque_upload_ids"`
	RawLocalPathsAccepted bool     `json:"raw_local_paths_accepted"`
	WorkspaceFileAttach   bool     `json:"workspace_file_attach"`
	MIMEAllowlist         []string `json:"mime_allowlist"`
	Retention             string   `json:"retention"`
}

type capabilityVoice struct {
	DeviceTranscribedTextTurns bool     `json:"device_transcribed_text_turns"`
	RawAudioUpload             bool     `json:"raw_audio_upload"`
	VoiceProfilesEndpoint      string   `json:"voice_profiles_endpoint"`
	RunRecordsEndpoint         string   `json:"run_records_endpoint"`
	STTProviders               []string `json:"stt_providers"`
	TTSProviders               []string `json:"tts_providers"`
}

type capabilityStreams struct {
	CanonicalEndpoint string   `json:"canonical_endpoint"`
	Transport         string   `json:"transport"`
	EventKinds        []string `json:"event_kinds"`
	OpenAIRunsBridge  bool     `json:"openai_runs_bridge"`
}

type capabilityDurableReconnect struct {
	Supported         bool     `json:"supported"`
	IssueEndpoint     string   `json:"issue_endpoint"`
	AuthMethods       []string `json:"auth_methods"`
	Platforms         []string `json:"platforms"`
	EffectiveSecurity string   `json:"effective_security"`
	BlockedReason     string   `json:"blocked_reason"`
}

type capabilityDocument struct {
	Object            string                      `json:"object"`
	ProtocolVersion   string                      `json:"protocol_version"`
	Capabilities      []string                    `json:"capabilities"`
	Auth              capabilityAuth              `json:"auth"`
	Health            capabilityHealth            `json:"health"`
	Endpoints         []capabilityEndpoint        `json:"endpoints"`
	ProfileManagement capabilityProfileManagement `json:"profile_management"`
	Attachments       capabilityAttachments       `json:"attachments"`
	Voice             capabilityVoice             `json:"voice"`
	Streams           capabilityStreams           `json:"streams"`
	DurableReconnect  capabilityDurableReconnect  `json:"durable_reconnect"`
}

func (c *Channel) handleCapabilities(w http.ResponseWriter, r *http.Request, _ string) {
	if r.Method != http.MethodGet {
		writeNavivoxError(w, http.StatusMethodNotAllowed, "", "bad_request", "Method not allowed")
		return
	}
	writeNavivoxJSON(w, http.StatusOK, c.capabilityDocumentForRequest(r))
}

func (c *Channel) capabilityDocument() capabilityDocument {
	return c.capabilityDocumentForRequest(nil)
}

func (c *Channel) capabilityDocumentForRequest(r *http.Request) capabilityDocument {
	matrix := navivoxVoiceProviderMatrix()
	events := navivoxEventKinds()
	authMode := strings.TrimSpace(c.cfg.AuthMode)
	return capabilityDocument{
		Object:          "gormes.navivox.capabilities",
		ProtocolVersion: navivoxWebSocketProtocol,
		Capabilities:    navivoxCapabilityNames(),
		Auth: capabilityAuth{
			Mode:               authMode,
			Headers:            navivoxCapabilityAuthHeaders(authMode),
			WebSocketProtocols: navivoxCapabilityWebSocketProtocols(authMode),
		},
		Health: capabilityHealth{
			Canonical: "/healthz",
			Aliases:   []string{"/healthz"},
			Auth:      "none",
		},
		Endpoints: navivoxCapabilityEndpoints(),
		ProfileManagement: capabilityProfileManagement{
			ContactsEndpoint:       "/v1/navivox/profile-contacts",
			RoutingEndpoint:        "/v1/navivox/profile-routing",
			CreateFromSeedEndpoint: "/v1/navivox/profile-seed",
			DashboardAPIExposed:    false,
			SupportedActions:       []string{"contact_snapshot", "contact_updates", "routing_read", "seed_draft", "create_from_seed"},
			UnsupportedActions:     []string{"direct_dashboard_api_profiles", "bulk_import", "raw_config_document", "raw_local_path_import"},
			ProfileContractParts:   []string{"profile_contacts", "profile_routing", "voice_profiles"},
		},
		Attachments: capabilityAttachments{
			MaxRequestBytes:       navivoxMaxTurnRequestBytes,
			OpaqueUploadIDs:       false,
			RawLocalPathsAccepted: false,
			WorkspaceFileAttach:   false,
			MIMEAllowlist:         []string{},
			Retention:             "not_accepted",
		},
		Voice: capabilityVoice{
			DeviceTranscribedTextTurns: true,
			RawAudioUpload:             false,
			VoiceProfilesEndpoint:      "/v1/navivox/voice-profiles",
			RunRecordsEndpoint:         "/v1/navivox/run-records/{run_id_or_session_id}",
			STTProviders:               matrix.STTProviders,
			TTSProviders:               matrix.TTSProviders,
		},
		Streams: capabilityStreams{
			CanonicalEndpoint: "/v1/navivox/stream",
			Transport:         "websocket",
			EventKinds:        events,
			OpenAIRunsBridge:  false,
		},
		DurableReconnect: navivoxDurableReconnectCapability(r, c.cfg),
	}
}

func navivoxDurableReconnectCapability(r *http.Request, cfg config.NavivoxCfg) capabilityDurableReconnect {
	transportSecurity := navivoxTransportSecurityStatusForRequest(r, cfg)
	return capabilityDurableReconnect{
		Supported:         false,
		IssueEndpoint:     "",
		AuthMethods:       []string{},
		Platforms:         []string{"android"},
		EffectiveSecurity: transportSecurity.EffectiveSecurity,
		BlockedReason:     "Durable credential issuance is not implemented yet.",
	}
}

func navivoxCapabilityAuthHeaders(mode string) []string {
	var headers []string
	if navivoxAuthModeUsesToken(mode) {
		headers = append(headers, "Authorization: Bearer <token>", "X-Gormes-Navivox-Token")
	}
	if navivoxAuthModeUsesTailscale(mode) {
		headers = append(headers, "Tailscale-User-Login", "X-Tailscale-User-Login", "Tailscale-Device-Name", "X-Tailscale-Device-Name")
	}
	return headers
}

func navivoxCapabilityWebSocketProtocols(mode string) []string {
	protocols := []string{navivoxWebSocketProtocol}
	if navivoxAuthModeUsesToken(mode) {
		protocols = append(protocols, navivoxWebSocketTokenProtocolPrefix+"<base64url-token>")
	}
	return protocols
}

func navivoxAuthModeUsesToken(mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case config.NavivoxAuthPairingToken, config.NavivoxAuthStaticToken, config.NavivoxAuthTokenAndTailscaleIdentity:
		return true
	default:
		return false
	}
}

func navivoxAuthModeUsesTailscale(mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case config.NavivoxAuthTailscaleIdentity, config.NavivoxAuthTokenAndTailscaleIdentity:
		return true
	default:
		return false
	}
}

func navivoxCapabilityNames() []string {
	return []string{
		"profile_contacts",
		"profile_routing",
		"profile_seed",
		"config_admin",
		"voice_profiles",
		"voice_run_records",
		"memory_overview",
		"stream_turns",
		"tool_progress",
		"safety_warnings",
		"approval_required",
		"turn_control",
	}
}

func navivoxEventKinds() []string {
	return []string{
		"pong",
		"session_started",
		"assistant_delta",
		"assistant_message",
		"tool_call_started",
		"tool_call_updated",
		"tool_call_finished",
		"safety_warning",
		"approval_required",
		"profile_contact_update",
		"error",
		"done",
	}
}

func navivoxCapabilityEndpoints() []capabilityEndpoint {
	return []capabilityEndpoint{
		{Method: http.MethodGet, Path: "/healthz", Auth: "none", Stability: "stable", Description: "Public liveness probe"},
		{Method: http.MethodGet, Path: "/v1/navivox/status", Auth: "navivox", Stability: "stable", Description: "Runtime status and lightweight capability list"},
		{Method: http.MethodGet, Path: "/v1/navivox/capabilities", Auth: "navivox", Stability: "stable", Description: "Versioned Navivox capability document"},
		{Method: http.MethodGet, Path: "/v1/navivox/profile-contacts", Auth: "navivox", Stability: "stable", Description: "Server-authoritative safe profile contact snapshot"},
		{Method: http.MethodGet, Path: "/v1/navivox/profile-routing", Auth: "navivox", Stability: "stable", Description: "Server/profile routing snapshot"},
		{Method: http.MethodPost, Path: "/v1/navivox/profile-seed", Auth: "navivox", Stability: "stable", Description: "Draft or apply a profile from operator text"},
		{Method: http.MethodGet, Path: "/v1/navivox/config-admin", Auth: "navivox", Stability: "stable", Description: "Safe config values"},
		{Method: http.MethodGet, Path: "/v1/navivox/config-admin/schema", Auth: "navivox", Stability: "stable", Description: "Safe config schema"},
		{Method: http.MethodPost, Path: "/v1/navivox/config-admin/diff", Auth: "navivox", Stability: "stable", Description: "Preview safe config changes"},
		{Method: http.MethodPost, Path: "/v1/navivox/config-admin/validate", Auth: "navivox", Stability: "stable", Description: "Validate safe config changes"},
		{Method: http.MethodPost, Path: "/v1/navivox/config-admin/apply", Auth: "navivox", Stability: "stable", Description: "Apply safe config changes"},
		{Method: http.MethodGet, Path: "/v1/navivox/voice-profiles", Auth: "navivox", Stability: "stable", Description: "Per-profile STT/TTS voice profile state"},
		{Method: http.MethodPost, Path: "/v1/navivox/voice-profiles/validate", Auth: "navivox", Stability: "stable", Description: "Validate per-profile voice profile settings"},
		{Method: http.MethodGet, Path: "/v1/navivox/run-records/{run_id_or_session_id}", Auth: "navivox", Stability: "stable", Description: "Voice and turn run record lookup"},
		{Method: http.MethodGet, Path: "/v1/navivox/memory/overview", Auth: "navivox", Stability: "stable", Description: "Bounded memory overview"},
		{Method: http.MethodGet, Path: "/v1/navivox/sessions", Auth: "navivox", Stability: "stable", Description: "Session list"},
		{Method: http.MethodGet, Path: "/v1/navivox/sessions/{session_id}", Auth: "navivox", Stability: "stable", Description: "Session snapshot"},
		{Method: http.MethodPost, Path: "/v1/navivox/turn", Auth: "navivox", Stability: "stable", Description: "Queue a text turn"},
		{Method: "WS", Path: "/v1/navivox/stream", Auth: "navivox", Stability: "stable", Description: "Canonical Navivox event stream"},
	}
}

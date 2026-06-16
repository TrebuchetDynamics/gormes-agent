package capability

import (
	"net/http"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

const (
	MaxTurnRequestBytes          = 1 << 20
	WebSocketProtocol            = "navivox.v1"
	WebSocketTokenProtocolPrefix = "gormes.navivox.token."
)

type Endpoint struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Auth        string `json:"auth"`
	Stability   string `json:"stability"`
	Description string `json:"description"`
}

type Auth struct {
	Mode               string   `json:"mode"`
	Headers            []string `json:"headers"`
	WebSocketProtocols []string `json:"websocket_protocols"`
}

type Health struct {
	Canonical string   `json:"canonical"`
	Aliases   []string `json:"aliases"`
	Auth      string   `json:"auth"`
}

type ProfileManagement struct {
	ContactsEndpoint       string   `json:"contacts_endpoint"`
	RoutingEndpoint        string   `json:"routing_endpoint"`
	CreateFromSeedEndpoint string   `json:"create_from_seed_endpoint"`
	DashboardAPIExposed    bool     `json:"dashboard_api_exposed"`
	SupportedActions       []string `json:"supported_actions"`
	UnsupportedActions     []string `json:"unsupported_actions"`
	ProfileContractParts   []string `json:"profile_contract_parts"`
}

type Attachments struct {
	MaxRequestBytes       int      `json:"max_request_bytes"`
	OpaqueUploadIDs       bool     `json:"opaque_upload_ids"`
	RawLocalPathsAccepted bool     `json:"raw_local_paths_accepted"`
	WorkspaceFileAttach   bool     `json:"workspace_file_attach"`
	MIMEAllowlist         []string `json:"mime_allowlist"`
	Retention             string   `json:"retention"`
}

type Voice struct {
	DeviceTranscribedTextTurns bool     `json:"device_transcribed_text_turns"`
	RawAudioUpload             bool     `json:"raw_audio_upload"`
	VoiceProfilesEndpoint      string   `json:"voice_profiles_endpoint"`
	RunRecordsEndpoint         string   `json:"run_records_endpoint"`
	STTProviders               []string `json:"stt_providers"`
	TTSProviders               []string `json:"tts_providers"`
}

type Streams struct {
	CanonicalEndpoint string   `json:"canonical_endpoint"`
	Transport         string   `json:"transport"`
	EventKinds        []string `json:"event_kinds"`
	OpenAIRunsBridge  bool     `json:"openai_runs_bridge"`
}

type DurableReconnectSchema struct {
	Supported         bool     `json:"supported"`
	IssueEndpoint     string   `json:"issue_endpoint"`
	AuthMethods       []string `json:"auth_methods"`
	Platforms         []string `json:"platforms"`
	EffectiveSecurity string   `json:"effective_security"`
	BlockedReason     string   `json:"blocked_reason"`
}

type DocumentSchema struct {
	Object            string                 `json:"object"`
	ProtocolVersion   string                 `json:"protocol_version"`
	Capabilities      []string               `json:"capabilities"`
	Auth              Auth                   `json:"auth"`
	Health            Health                 `json:"health"`
	Endpoints         []Endpoint             `json:"endpoints"`
	ProfileManagement ProfileManagement      `json:"profile_management"`
	Attachments       Attachments            `json:"attachments"`
	Voice             Voice                  `json:"voice"`
	Streams           Streams                `json:"streams"`
	DurableReconnect  DurableReconnectSchema `json:"durable_reconnect"`
}

type DocumentParams struct {
	ProtocolVersion   string
	AuthMode          string
	STTProviders      []string
	TTSProviders      []string
	EffectiveSecurity string
}

func Document(params DocumentParams) DocumentSchema {
	authMode := strings.TrimSpace(params.AuthMode)
	return DocumentSchema{
		Object:          "gormes.navivox.capabilities",
		ProtocolVersion: params.ProtocolVersion,
		Capabilities:    CapabilityNames(),
		Auth: Auth{
			Mode:               authMode,
			Headers:            AuthHeaders(authMode),
			WebSocketProtocols: WebSocketProtocols(authMode),
		},
		Health: Health{
			Canonical: "/healthz",
			Aliases:   []string{"/healthz"},
			Auth:      "none",
		},
		Endpoints: Endpoints(),
		ProfileManagement: ProfileManagement{
			ContactsEndpoint:       "/v1/navivox/profile-contacts",
			RoutingEndpoint:        "/v1/navivox/profile-routing",
			CreateFromSeedEndpoint: "/v1/navivox/profile-seed",
			DashboardAPIExposed:    false,
			SupportedActions:       []string{"contact_snapshot", "contact_updates", "routing_read", "seed_draft", "create_from_seed"},
			UnsupportedActions:     []string{"direct_dashboard_api_profiles", "bulk_import", "raw_config_document", "raw_local_path_import"},
			ProfileContractParts:   []string{"profile_contacts", "profile_routing", "voice_profiles"},
		},
		Attachments: Attachments{
			MaxRequestBytes:       MaxTurnRequestBytes,
			OpaqueUploadIDs:       false,
			RawLocalPathsAccepted: false,
			WorkspaceFileAttach:   false,
			MIMEAllowlist:         []string{},
			Retention:             "not_accepted",
		},
		Voice: Voice{
			DeviceTranscribedTextTurns: true,
			RawAudioUpload:             false,
			VoiceProfilesEndpoint:      "/v1/navivox/voice-profiles",
			RunRecordsEndpoint:         "/v1/navivox/run-records/{run_id_or_session_id}",
			STTProviders:               params.STTProviders,
			TTSProviders:               params.TTSProviders,
		},
		Streams: Streams{
			CanonicalEndpoint: "/v1/navivox/stream",
			Transport:         "websocket",
			EventKinds:        EventKinds(),
			OpenAIRunsBridge:  false,
		},
		DurableReconnect: DurableReconnect(params.EffectiveSecurity),
	}
}

func DurableReconnect(effectiveSecurity string) DurableReconnectSchema {
	return DurableReconnectSchema{
		Supported:         false,
		IssueEndpoint:     "",
		AuthMethods:       []string{},
		Platforms:         []string{"android"},
		EffectiveSecurity: effectiveSecurity,
		BlockedReason:     "Durable credential issuance is not implemented yet.",
	}
}

func AuthHeaders(mode string) []string {
	var headers []string
	if AuthModeUsesToken(mode) {
		headers = append(headers, "Authorization: Bearer <token>", "X-Gormes-Navivox-Token")
	}
	if AuthModeUsesTailscale(mode) {
		headers = append(headers, "Tailscale-User-Login", "X-Tailscale-User-Login", "Tailscale-Device-Name", "X-Tailscale-Device-Name")
	}
	return headers
}

func WebSocketProtocols(mode string) []string {
	protocols := []string{WebSocketProtocol}
	if AuthModeUsesToken(mode) {
		protocols = append(protocols, WebSocketTokenProtocolPrefix+"<base64url-token>")
	}
	return protocols
}

func AuthModeUsesToken(mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case config.NavivoxAuthPairingToken, config.NavivoxAuthStaticToken, config.NavivoxAuthTokenAndTailscaleIdentity:
		return true
	default:
		return false
	}
}

func AuthModeUsesTailscale(mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case config.NavivoxAuthTailscaleIdentity, config.NavivoxAuthTokenAndTailscaleIdentity:
		return true
	default:
		return false
	}
}

func CapabilityNames() []string {
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

func EventKinds() []string {
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

func Endpoints() []Endpoint {
	return []Endpoint{
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

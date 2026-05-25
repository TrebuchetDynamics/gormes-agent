package navivox

import (
	"net/http"
	"strings"
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
	AcceptedModes      []string `json:"accepted_modes"`
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
	PayloadSafety          []string `json:"payload_safety"`
}

type capabilityAttachments struct {
	MaxRequestBytes          int      `json:"max_request_bytes"`
	OpaqueUploadIDs          bool     `json:"opaque_upload_ids"`
	RawLocalPathsAccepted    bool     `json:"raw_local_paths_accepted"`
	WorkspaceFileAttach      bool     `json:"workspace_file_attach"`
	MIMEAllowlist            []string `json:"mime_allowlist"`
	Retention                string   `json:"retention"`
	UnsupportedUntilEndpoint string   `json:"unsupported_until_endpoint"`
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
	RunsBridge        string   `json:"runs_bridge"`
}

type capabilityDeprecation struct {
	Surface     string `json:"surface"`
	Rule        string `json:"rule"`
	Replacement string `json:"replacement"`
}

type capabilityDocument struct {
	Object             string                      `json:"object"`
	ProtocolVersion    string                      `json:"protocol_version"`
	Capabilities       []string                    `json:"capabilities"`
	Auth               capabilityAuth              `json:"auth"`
	Health             capabilityHealth            `json:"health"`
	Endpoints          []capabilityEndpoint        `json:"endpoints"`
	Events             []string                    `json:"events"`
	ProfileManagement  capabilityProfileManagement `json:"profile_management"`
	Attachments        capabilityAttachments       `json:"attachments"`
	Voice              capabilityVoice             `json:"voice"`
	Streams            capabilityStreams           `json:"streams"`
	Deprecations       []capabilityDeprecation     `json:"deprecations"`
	CompatibilityNotes []string                    `json:"compatibility_notes"`
}

func (c *Channel) handleCapabilities(w http.ResponseWriter, r *http.Request, _ string) {
	if r.Method != http.MethodGet {
		writeNavivoxError(w, http.StatusMethodNotAllowed, "", "bad_request", "Method not allowed")
		return
	}
	writeNavivoxJSON(w, http.StatusOK, c.capabilityDocument())
}

func (c *Channel) capabilityDocument() capabilityDocument {
	matrix := navivoxVoiceProviderMatrix()
	events := navivoxEventKinds()
	return capabilityDocument{
		Object:          "gormes.navivox.capabilities",
		ProtocolVersion: navivoxWebSocketProtocol,
		Capabilities:    navivoxCapabilityNames(),
		Auth: capabilityAuth{
			Mode:               strings.TrimSpace(c.cfg.AuthMode),
			AcceptedModes:      []string{"pairing_token", "static_token", "tailscale_identity", "token_and_tailscale_identity"},
			Headers:            []string{"Authorization: Bearer <token>", "X-Gormes-Navivox-Token", "Tailscale-User-Login", "Tailscale-Device-Name"},
			WebSocketProtocols: []string{navivoxWebSocketProtocol, navivoxLegacyWebSocketProtocol, navivoxWebSocketTokenProtocolPrefix + "<base64url-token>"},
		},
		Health: capabilityHealth{
			Canonical: "/healthz",
			Aliases:   []string{"/healthz"},
			Auth:      "none",
		},
		Endpoints: navivoxCapabilityEndpoints(),
		Events:    events,
		ProfileManagement: capabilityProfileManagement{
			ContactsEndpoint:       "/v1/navivox/profile-contacts",
			RoutingEndpoint:        "/v1/navivox/profile-routing",
			CreateFromSeedEndpoint: "/v1/navivox/profile-seed",
			DashboardAPIExposed:    false,
			SupportedActions:       []string{"contact_snapshot", "contact_updates", "routing_read", "seed_draft", "create_from_seed"},
			UnsupportedActions:     []string{"direct_dashboard_api_profiles", "bulk_import", "raw_config_document", "raw_local_path_import"},
			PayloadSafety:          []string{"profile_id", "display_name", "health", "attention_badges", "workspace_status", "routing_options", "voice_capability"},
		},
		Attachments: capabilityAttachments{
			MaxRequestBytes:          navivoxMaxTurnRequestBytes,
			OpaqueUploadIDs:          false,
			RawLocalPathsAccepted:    false,
			WorkspaceFileAttach:      false,
			MIMEAllowlist:            []string{},
			Retention:                "not_accepted",
			UnsupportedUntilEndpoint: "/v1/navivox/uploads",
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
			RunsBridge:        "navivox_stream_is_canonical_for_navivox_clients",
		},
		Deprecations: []capabilityDeprecation{
			{Surface: "dashboard_profiles", Rule: "do_not_call_from_navivox_clients", Replacement: "/v1/navivox/profile-contacts and /v1/navivox/profile-seed"},
			{Surface: navivoxLegacyWebSocketProtocol, Rule: "accepted_for_legacy_clients", Replacement: navivoxWebSocketProtocol},
			{Surface: "local_file_attachment_paths", Rule: "not_accepted", Replacement: "future opaque upload ids"},
		},
		CompatibilityNotes: []string{
			"Navivox clients should enable UI affordances only when this document advertises the required endpoint and action.",
			"/v1/navivox/stream is the canonical Navivox event stream; /v1/runs remains the OpenAI-style API server surface.",
		},
	}
}

func navivoxCapabilityNames() []string {
	return []string{
		"capability_document",
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
		"setup_handoff",
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

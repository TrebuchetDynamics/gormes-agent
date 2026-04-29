package acp

import (
	"path"
	"sort"
	"strings"
)

type Surface string

const (
	SurfaceAuth        Surface = "auth"
	SurfaceEntry       Surface = "entry"
	SurfaceSession     Surface = "session"
	SurfaceTools       Surface = "tools"
	SurfacePermissions Surface = "permissions"
	SurfaceEvents      Surface = "events"
	SurfaceServer      Surface = "server"
	SurfaceRegistry    Surface = "registry"
)

type ServerSurfaceStatus string

const (
	ServerSurfaceStatusImplemented ServerSurfaceStatus = "implemented"
	ServerSurfaceStatusRowBacked   ServerSurfaceStatus = "row_backed"
	ServerSurfaceStatusOwned       ServerSurfaceStatus = "owned"
	ServerSurfaceStatusExcluded    ServerSurfaceStatus = "excluded"
)

const ServerSurfaceRowBackedEvidence = "acp_server_surface_row_backed"

type ServerSurfaceRow struct {
	Surface        Surface
	UpstreamFile   string
	Status         ServerSurfaceStatus
	TargetPackage  string
	ProgressRow    string
	EvidenceCode   string
	PlannedSymbols []string
}

type ServerManifest struct {
	Surfaces []ServerSurfaceRow
}

func DefaultServerManifest() ServerManifest {
	rows := []ServerSurfaceRow{
		rowBacked(SurfaceAuth, "acp_adapter/auth.py",
			"DetectProvider", "HasProvider"),
		rowBacked(SurfaceEntry, "acp_adapter/entry.py",
			"RunStdioEntry", "BenignProbeMethods"),
		rowBacked(SurfaceSession, "acp_adapter/session.py",
			"SessionManager", "SessionState", "NormalizeCWD"),
		rowBacked(SurfaceTools, "acp_adapter/tools.py",
			"ToolKindFor", "ToolCallStart", "ToolCallProgress", "ToolCallComplete"),
		rowBacked(SurfacePermissions, "acp_adapter/permissions.py",
			"ApprovalCallback", "PermissionOption", "PermissionOutcomeForKind"),
		rowBacked(SurfaceEvents, "acp_adapter/events.py",
			"SessionUpdate", "ToolProgressEmitter", "ThinkingEmitter", "MessageEmitter"),
		rowBacked(SurfaceServer, "acp_adapter/server.py",
			"HermesACPAgent", "AgentCapabilities", "InitializeResponse"),
		rowBacked(SurfaceRegistry, "acp_registry/agent.json",
			"AgentRegistryEntry", "AgentRegistryDistribution"),
	}
	sort.Slice(rows, func(i, j int) bool { return string(rows[i].Surface) < string(rows[j].Surface) })
	return ServerManifest{Surfaces: rows}
}

func (m ServerManifest) Lookup(surface Surface) (ServerSurfaceRow, bool) {
	for _, row := range m.Surfaces {
		if row.Surface == surface {
			return row, true
		}
	}
	return ServerSurfaceRow{}, false
}

func (m ServerManifest) MissingUpstreamFiles(upstream []string) []string {
	known := make(map[string]struct{}, len(m.Surfaces))
	for _, row := range m.Surfaces {
		known[normalizeUpstreamFile(row.UpstreamFile)] = struct{}{}
	}
	missing := make([]string, 0)
	for _, file := range upstream {
		norm := normalizeUpstreamFile(file)
		if !isACPUpstreamFile(norm) {
			continue
		}
		if _, ok := known[norm]; ok {
			continue
		}
		missing = append(missing, norm)
	}
	sort.Strings(missing)
	return missing
}

func rowBacked(surface Surface, upstreamFile string, symbols ...string) ServerSurfaceRow {
	return ServerSurfaceRow{
		Surface:        surface,
		UpstreamFile:   upstreamFile,
		Status:         ServerSurfaceStatusRowBacked,
		TargetPackage:  "internal/acp",
		ProgressRow:    "5.H ACP server side",
		EvidenceCode:   ServerSurfaceRowBackedEvidence,
		PlannedSymbols: append([]string(nil), symbols...),
	}
}

func normalizeUpstreamFile(file string) string {
	file = strings.ReplaceAll(file, "\\", "/")
	return path.Clean(file)
}

func isACPUpstreamFile(file string) bool {
	if strings.HasPrefix(file, "acp_adapter/") && strings.HasSuffix(file, ".py") {
		base := path.Base(file)
		return base != "__init__.py" && base != "__main__.py"
	}
	if file == "acp_registry/agent.json" {
		return true
	}
	return false
}

type AgentRegistryDistribution struct {
	Type    string
	Command string
	Args    []string
}

type AgentRegistryEntry struct {
	SchemaVersion int
	Name          string
	DisplayName   string
	Description   string
	Icon          string
	Distribution  AgentRegistryDistribution
}

func DefaultAgentRegistryEntry() AgentRegistryEntry {
	return AgentRegistryEntry{
		SchemaVersion: 1,
		Name:          "hermes-agent",
		DisplayName:   "Hermes Agent",
		Description:   "AI agent by Nous Research with 90+ tools, persistent memory, and multi-platform support",
		Icon:          "icon.svg",
		Distribution: AgentRegistryDistribution{
			Type:    "command",
			Command: "hermes",
			Args:    []string{"acp"},
		},
	}
}

type PermissionOptionKind string

const (
	PermissionAllowOnce    PermissionOptionKind = "allow_once"
	PermissionAllowAlways  PermissionOptionKind = "allow_always"
	PermissionRejectOnce   PermissionOptionKind = "reject_once"
	PermissionRejectAlways PermissionOptionKind = "reject_always"
)

func PermissionOutcomeForKind(kind PermissionOptionKind) string {
	switch kind {
	case PermissionAllowOnce:
		return "once"
	case PermissionAllowAlways:
		return "always"
	case PermissionRejectOnce, PermissionRejectAlways:
		return "deny"
	default:
		return "deny"
	}
}

type ToolKind string

const (
	ToolKindRead    ToolKind = "read"
	ToolKindEdit    ToolKind = "edit"
	ToolKindSearch  ToolKind = "search"
	ToolKindExecute ToolKind = "execute"
	ToolKindFetch   ToolKind = "fetch"
	ToolKindThink   ToolKind = "think"
	ToolKindOther   ToolKind = "other"
)

var toolKindMap = map[string]ToolKind{
	"read_file":         ToolKindRead,
	"write_file":        ToolKindEdit,
	"patch":             ToolKindEdit,
	"search_files":      ToolKindSearch,
	"terminal":          ToolKindExecute,
	"process":           ToolKindExecute,
	"execute_code":      ToolKindExecute,
	"web_search":        ToolKindFetch,
	"web_extract":       ToolKindFetch,
	"browser_navigate":  ToolKindFetch,
	"browser_click":     ToolKindExecute,
	"browser_type":      ToolKindExecute,
	"browser_snapshot":  ToolKindRead,
	"browser_vision":    ToolKindRead,
	"browser_scroll":    ToolKindExecute,
	"browser_press":     ToolKindExecute,
	"browser_back":      ToolKindExecute,
	"browser_get_images": ToolKindRead,
	"delegate_task":     ToolKindExecute,
	"vision_analyze":    ToolKindRead,
	"image_generate":    ToolKindExecute,
	"text_to_speech":    ToolKindExecute,
	"_thinking":         ToolKindThink,
}

func ToolKindFor(name string) ToolKind {
	if kind, ok := toolKindMap[name]; ok {
		return kind
	}
	return ToolKindOther
}

type ToolCallStart struct {
	ID   string
	Name string
	Kind ToolKind
}

type SessionUpdateKind string

const (
	SessionUpdateToolStart    SessionUpdateKind = "tool_call_start"
	SessionUpdateToolComplete SessionUpdateKind = "tool_call_complete"
	SessionUpdateAgentThought SessionUpdateKind = "agent_thought_text"
	SessionUpdateAgentMessage SessionUpdateKind = "agent_message_text"
)

type SessionUpdate struct {
	Kind     SessionUpdateKind
	ToolCall *ToolCallStart
	Text     string
}

type SessionPhase string

const (
	SessionPhaseInitialize   SessionPhase = "initialize"
	SessionPhaseAuthenticate SessionPhase = "authenticate"
	SessionPhaseNew          SessionPhase = "new_session"
	SessionPhaseLoad         SessionPhase = "load_session"
	SessionPhaseResume       SessionPhase = "resume_session"
	SessionPhasePrompt       SessionPhase = "prompt"
	SessionPhaseCancel       SessionPhase = "cancel"
)

func DefaultSessionLifecycle() []SessionPhase {
	return []SessionPhase{
		SessionPhaseInitialize,
		SessionPhaseAuthenticate,
		SessionPhaseNew,
		SessionPhaseLoad,
		SessionPhaseResume,
		SessionPhasePrompt,
		SessionPhaseCancel,
	}
}

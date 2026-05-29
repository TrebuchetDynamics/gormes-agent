package acp

import (
	"reflect"
	"sort"
	"testing"
)

func TestACPServerManifest_ClassifiesUpstreamSurfaces(t *testing.T) {
	manifest := DefaultServerManifest()

	type expectation struct {
		surface       Surface
		upstreamFile  string
		status        ServerSurfaceStatus
		targetPackage string
		evidenceCode  string
		mustHaveSym   string
	}

	expectations := []expectation{
		{SurfaceAuth, "acp_adapter/auth.py", ServerSurfaceStatusRowBacked, "internal/acp", ServerSurfaceRowBackedEvidence, "DetectProvider"},
		{SurfaceEntry, "acp_adapter/entry.py", ServerSurfaceStatusRowBacked, "internal/acp", ServerSurfaceRowBackedEvidence, "RunStdioEntry"},
		{SurfaceSession, "acp_adapter/session.py", ServerSurfaceStatusRowBacked, "internal/acp", ServerSurfaceRowBackedEvidence, "SessionManager"},
		{SurfaceTools, "acp_adapter/tools.py", ServerSurfaceStatusRowBacked, "internal/acp", ServerSurfaceRowBackedEvidence, "ToolKindFor"},
		{SurfacePermissions, "acp_adapter/permissions.py", ServerSurfaceStatusRowBacked, "internal/acp", ServerSurfaceRowBackedEvidence, "ApprovalCallback"},
		{SurfaceEvents, "acp_adapter/events.py", ServerSurfaceStatusRowBacked, "internal/acp", ServerSurfaceRowBackedEvidence, "SessionUpdate"},
		{SurfaceServer, "acp_adapter/server.py", ServerSurfaceStatusRowBacked, "internal/acp", ServerSurfaceRowBackedEvidence, "HermesACPAgent"},
		{SurfaceRegistry, "acp_registry/agent.json", ServerSurfaceStatusRowBacked, "internal/acp", ServerSurfaceRowBackedEvidence, "AgentRegistryEntry"},
	}

	for _, want := range expectations {
		t.Run(string(want.surface), func(t *testing.T) {
			row, ok := manifest.Lookup(want.surface)
			if !ok {
				t.Fatalf("manifest missing surface %q", want.surface)
			}
			if row.UpstreamFile != want.upstreamFile {
				t.Fatalf("upstream file = %q, want %q", row.UpstreamFile, want.upstreamFile)
			}
			if row.Status != want.status {
				t.Fatalf("status = %q, want %q", row.Status, want.status)
			}
			if row.TargetPackage != want.targetPackage {
				t.Fatalf("target package = %q, want %q", row.TargetPackage, want.targetPackage)
			}
			if row.EvidenceCode != want.evidenceCode {
				t.Fatalf("evidence code = %q, want %q", row.EvidenceCode, want.evidenceCode)
			}
			if row.ProgressRow != "5.H ACP server side" {
				t.Fatalf("progress row = %q, want 5.H ACP server side", row.ProgressRow)
			}
			found := false
			for _, sym := range row.PlannedSymbols {
				if sym == want.mustHaveSym {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("planned symbols %v missing %q", row.PlannedSymbols, want.mustHaveSym)
			}
		})
	}
}

func TestACPServerManifest_RejectsUnknownUpstreamFiles(t *testing.T) {
	manifest := DefaultServerManifest()

	upstream := []string{
		"acp_adapter/auth.py",
		"acp_adapter/entry.py",
		"acp_adapter/events.py",
		"acp_adapter/permissions.py",
		"acp_adapter/server.py",
		"acp_adapter/session.py",
		"acp_adapter/tools.py",
		"acp_registry/agent.json",
	}
	if missing := manifest.MissingUpstreamFiles(upstream); len(missing) != 0 {
		t.Fatalf("missing upstream files = %v, want none", missing)
	}

	drift := []string{
		"acp_adapter/auth.py",
		"acp_adapter/transport.py",
		"acp_registry/agent.json",
		"acp_registry/icon.svg",
	}
	got := manifest.MissingUpstreamFiles(drift)
	want := []string{"acp_adapter/transport.py"}
	sort.Strings(got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MissingUpstreamFiles drift = %v, want %v", got, want)
	}
}

func TestACPServerManifest_NoLiveACPSideEffects(t *testing.T) {
	manifest := DefaultServerManifest()
	for _, row := range manifest.Surfaces {
		if row.Status == ServerSurfaceStatusImplemented {
			t.Fatalf("surface %q claims implemented before live ACP rows; status=%q", row.Surface, row.Status)
		}
		if row.Status != ServerSurfaceStatusRowBacked &&
			row.Status != ServerSurfaceStatusOwned &&
			row.Status != ServerSurfaceStatusExcluded {
			t.Fatalf("surface %q has unsupported status %q", row.Surface, row.Status)
		}
	}
}

func TestAgentRegistryEntry_MatchesUpstreamShape(t *testing.T) {
	entry := DefaultAgentRegistryEntry()
	if entry.SchemaVersion != 1 {
		t.Fatalf("schema_version = %d, want 1", entry.SchemaVersion)
	}
	if entry.Name != "hermes-agent" {
		t.Fatalf("name = %q, want hermes-agent", entry.Name)
	}
	if entry.DisplayName == "" {
		t.Fatal("display_name is empty")
	}
	if entry.Distribution.Type != "command" {
		t.Fatalf("distribution.type = %q, want command", entry.Distribution.Type)
	}
	if entry.Distribution.Command != "hermes" {
		t.Fatalf("distribution.command = %q, want hermes", entry.Distribution.Command)
	}
	if !reflect.DeepEqual(entry.Distribution.Args, []string{"acp"}) {
		t.Fatalf("distribution.args = %#v, want [acp]", entry.Distribution.Args)
	}
}

func TestPermissionOutcome_MapsACPKindToHermes(t *testing.T) {
	tests := []struct {
		kind PermissionOptionKind
		want string
	}{
		{PermissionAllowOnce, "once"},
		{PermissionAllowAlways, "always"},
		{PermissionRejectOnce, "deny"},
		{PermissionRejectAlways, "deny"},
	}
	for _, tt := range tests {
		got := PermissionOutcomeForKind(tt.kind)
		if got != tt.want {
			t.Fatalf("PermissionOutcomeForKind(%q) = %q, want %q", tt.kind, got, tt.want)
		}
	}
	if PermissionOutcomeForKind("unknown_kind") != "deny" {
		t.Fatal("unknown kinds must fall back to deny")
	}
}

func TestToolKindFor_MapsHermesToolNames(t *testing.T) {
	cases := map[string]ToolKind{
		"read_file":    ToolKindRead,
		"write_file":   ToolKindEdit,
		"patch":        ToolKindEdit,
		"search_files": ToolKindSearch,
		"terminal":     ToolKindExecute,
		"web_search":   ToolKindFetch,
		"_thinking":    ToolKindThink,
		"unknown_tool": ToolKindOther,
	}
	for tool, want := range cases {
		got := ToolKindFor(tool)
		if got != want {
			t.Fatalf("ToolKindFor(%q) = %q, want %q", tool, got, want)
		}
	}
}

func TestSessionUpdate_FixtureShapeIsTyped(t *testing.T) {
	start := SessionUpdate{
		Kind:     SessionUpdateToolStart,
		ToolCall: &ToolCallStart{ID: "tc-abc", Name: "read_file", Kind: ToolKindRead},
	}
	if start.Kind != SessionUpdateToolStart {
		t.Fatalf("kind = %q", start.Kind)
	}
	if start.ToolCall == nil || start.ToolCall.Name != "read_file" {
		t.Fatalf("ToolCallStart not preserved: %+v", start.ToolCall)
	}

	thought := SessionUpdate{Kind: SessionUpdateAgentThought, Text: "thinking"}
	if thought.Kind != SessionUpdateAgentThought || thought.Text != "thinking" {
		t.Fatalf("agent thought update malformed: %+v", thought)
	}
}

func TestSessionLifecycle_PhasesAreOrdered(t *testing.T) {
	want := []SessionPhase{
		SessionPhaseInitialize,
		SessionPhaseAuthenticate,
		SessionPhaseNew,
		SessionPhaseLoad,
		SessionPhaseResume,
		SessionPhasePrompt,
		SessionPhaseCancel,
	}
	got := DefaultSessionLifecycle()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DefaultSessionLifecycle() = %v, want %v", got, want)
	}
}

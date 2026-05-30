package mcp

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/access"
)

// TestToolDeclaration_RendersProviderSchemaAndMCPMetadata proves that a single
// ToolDeclaration source struct renders both the provider JSON-schema view
// (matching the existing tool registry shape) and the MCP metadata view, so
// the boundary keeps one source of truth.
func TestToolDeclaration_RendersProviderSchemaAndMCPMetadata(t *testing.T) {
	t.Parallel()

	decl := ToolDeclaration{
		ServerName:  "honcho",
		ToolName:    "honcho_chat",
		Description: "Reply on behalf of a peer with Honcho memory",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"peer": map[string]any{"type": "string"},
			},
			"required": []any{"peer"},
		},
	}

	provider := decl.ProviderSchema()
	if provider == nil {
		t.Fatal("ProviderSchema returned nil")
	}
	gotName, _ := provider["name"].(string)
	if gotName != decl.ToolName {
		t.Errorf("ProviderSchema name = %q, want %q", gotName, decl.ToolName)
	}
	gotDesc, _ := provider["description"].(string)
	if gotDesc != decl.Description {
		t.Errorf("ProviderSchema description = %q, want %q", gotDesc, decl.Description)
	}
	gotParams, ok := provider["parameters"]
	if !ok {
		t.Fatalf("ProviderSchema missing parameters field; got %v", provider)
	}
	if !reflect.DeepEqual(gotParams, decl.InputSchema) {
		t.Errorf("ProviderSchema parameters = %v, want %v", gotParams, decl.InputSchema)
	}

	// Provider schema must round-trip as JSON shaped like the existing
	// ToolDescriptor function envelope (name/description/parameters).
	raw, err := json.Marshal(provider)
	if err != nil {
		t.Fatalf("marshal provider schema: %v", err)
	}
	var probe struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Parameters  json.RawMessage `json:"parameters"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		t.Fatalf("unmarshal provider schema: %v", err)
	}
	if probe.Name != decl.ToolName || probe.Description != decl.Description || len(probe.Parameters) == 0 {
		t.Errorf("provider schema JSON shape mismatch: %+v", probe)
	}

	meta := decl.MCPMetadata()
	if meta.Name != decl.ToolName {
		t.Errorf("MCPMetadata.Name = %q, want %q", meta.Name, decl.ToolName)
	}
	if meta.ServerName != decl.ServerName {
		t.Errorf("MCPMetadata.ServerName = %q, want %q", meta.ServerName, decl.ServerName)
	}
	if meta.Description != decl.Description {
		t.Errorf("MCPMetadata.Description = %q, want %q", meta.Description, decl.Description)
	}
	if !reflect.DeepEqual(meta.InputSchema, decl.InputSchema) {
		t.Errorf("MCPMetadata.InputSchema = %v, want %v", meta.InputSchema, decl.InputSchema)
	}

	// Mutating the rendered provider schema must not affect the source
	// declaration: defensive copy proves the single-source guarantee.
	provider["name"] = "tampered"
	if decl.ToolName == "tampered" {
		t.Error("ProviderSchema returned shared reference; mutation leaked into ToolDeclaration")
	}
}

// TestToolFilter_ChannelTrustToolsetRestriction proves that include/exclude
// filters restrict tools by channel, trust class, and configured toolset.
func TestToolFilter_ChannelTrustToolsetRestriction(t *testing.T) {
	t.Parallel()

	all := ToolDeclaration{
		ServerName: "core",
		ToolName:   "all_tool",
	}
	scoped := ToolDeclaration{
		ServerName: "core",
		ToolName:   "scoped_tool",
		Channels:   []string{"slack"},
		TrustClass: []access.TrustClass{access.TrustClassOperator},
		Toolset:    []string{"ops"},
	}
	other := ToolDeclaration{
		ServerName: "core",
		ToolName:   "other_tool",
		Channels:   []string{"discord"},
		TrustClass: []access.TrustClass{access.TrustClassSystem},
		Toolset:    []string{"infra"},
	}

	filterMatch := ToolFilter{
		Channel:    "slack",
		TrustClass: access.TrustClassOperator,
		Toolset:    []string{"ops", "extras"},
	}
	if !filterMatch.Allows(all) {
		t.Error("filter must allow declaration with no scoping (all-channels/all-trust/always)")
	}
	if !filterMatch.Allows(scoped) {
		t.Error("filter must allow scoped declaration when channel/trust/toolset all match")
	}
	if filterMatch.Allows(other) {
		t.Error("filter must exclude declaration on a different channel/trust/toolset")
	}

	// Channel mismatch alone is enough to exclude.
	wrongChannel := ToolFilter{
		Channel:    "discord",
		TrustClass: access.TrustClassOperator,
		Toolset:    []string{"ops"},
	}
	if wrongChannel.Allows(scoped) {
		t.Error("channel mismatch must exclude scoped declaration")
	}

	// Trust mismatch alone is enough to exclude.
	wrongTrust := ToolFilter{
		Channel:    "slack",
		TrustClass: access.TrustClassChildAgent,
		Toolset:    []string{"ops"},
	}
	if wrongTrust.Allows(scoped) {
		t.Error("trust class mismatch must exclude scoped declaration")
	}

	// Toolset mismatch alone is enough to exclude.
	wrongToolset := ToolFilter{
		Channel:    "slack",
		TrustClass: access.TrustClassOperator,
		Toolset:    []string{"infra"},
	}
	if wrongToolset.Allows(scoped) {
		t.Error("toolset mismatch must exclude scoped declaration")
	}
}

// TestMCPHostAudit_RecordsServerToolNameAndRedactionStatus proves that audit
// events emitted by the host wrapper carry server name, tool name,
// args-redaction status, and result status (ok / unavailable / error).
func TestMCPHostAudit_RecordsServerToolNameAndRedactionStatus(t *testing.T) {
	t.Parallel()

	host := newFakeMCPHost()
	host.tools["honcho_chat"] = fakeMCPTool{
		decl: ToolDeclaration{
			ServerName:  "honcho",
			ToolName:    "honcho_chat",
			Description: "fake",
		},
		result: Result{Status: ResultStatusOK, Body: []byte(`{"ok":true}`)},
	}
	auditor := &fakeMCPAuditor{}
	filter := ToolFilter{Channel: "slack", TrustClass: access.TrustClassOperator}

	res := RunFiltered(context.Background(), host, filter, auditor,
		"honcho", "honcho_chat",
		map[string]any{"peer": "alice", "secret_token": "xyzzy12345"},
		true, // redactArgs
	)
	if res.Status != ResultStatusOK {
		t.Fatalf("expected ok, got %q (%s)", res.Status, res.Reason)
	}
	if len(auditor.events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(auditor.events))
	}
	ev := auditor.events[0]
	if ev.Server != "honcho" || ev.Tool != "honcho_chat" {
		t.Errorf("event server/tool = %q/%q, want honcho/honcho_chat", ev.Server, ev.Tool)
	}
	if !ev.ArgsRedacted {
		t.Error("expected ArgsRedacted=true when redactArgs is set")
	}
	if ev.Status != ResultStatusOK {
		t.Errorf("event status = %q, want ok", ev.Status)
	}
	if ev.Unavailable {
		t.Error("ok event must not be marked unavailable")
	}

	// Error case: tool returns error status.
	host.tools["broken"] = fakeMCPTool{
		decl: ToolDeclaration{
			ServerName: "honcho",
			ToolName:   "broken",
		},
		result: Result{Status: ResultStatusError, Reason: "boom"},
	}
	auditor.events = nil
	_ = RunFiltered(context.Background(), host, filter, auditor,
		"honcho", "broken",
		map[string]any{"k": "v"},
		false,
	)
	if len(auditor.events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(auditor.events))
	}
	ev = auditor.events[0]
	if ev.Status != ResultStatusError {
		t.Errorf("error event status = %q, want error", ev.Status)
	}
	if ev.ArgsRedacted {
		t.Error("ArgsRedacted should be false when redactArgs is false")
	}
}

// TestMCPHostAudit_UnavailableEvidence proves that unavailable MCP servers
// produce mcp_host_unavailable evidence in the audit event and do not leak
// secret/argument substrings into the rendered Reason.
func TestMCPHostAudit_UnavailableEvidence(t *testing.T) {
	t.Parallel()

	host := newFakeMCPHost()
	host.tools["awol"] = fakeMCPTool{
		decl: ToolDeclaration{
			ServerName: "honcho",
			ToolName:   "awol",
		},
		result: Result{
			Status: ResultStatusUnavailable,
			Reason: "server offline",
		},
	}
	auditor := &fakeMCPAuditor{}
	filter := ToolFilter{Channel: "slack", TrustClass: access.TrustClassOperator}

	secretValue := "supersecretvalue9876"
	res := RunFiltered(context.Background(), host, filter, auditor,
		"honcho", "awol",
		map[string]any{"token": secretValue, "peer": "alice"},
		true,
	)
	if res.Status != ResultStatusUnavailable {
		t.Fatalf("expected unavailable, got %q", res.Status)
	}
	if !strings.Contains(res.Reason, HostUnavailableEvidence) {
		t.Errorf("reason must contain unavailable evidence %q, got %q", HostUnavailableEvidence, res.Reason)
	}
	if strings.Contains(res.Reason, secretValue) {
		t.Errorf("reason leaked secret value: %q", res.Reason)
	}
	if strings.Contains(res.Reason, "alice") {
		t.Errorf("reason leaked argument value: %q", res.Reason)
	}

	if len(auditor.events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(auditor.events))
	}
	ev := auditor.events[0]
	if !ev.Unavailable {
		t.Error("audit event must be marked Unavailable")
	}
	if ev.Status != ResultStatusUnavailable {
		t.Errorf("event status = %q, want unavailable", ev.Status)
	}
	if !strings.Contains(ev.Reason, HostUnavailableEvidence) {
		t.Errorf("audit reason missing unavailable evidence: %q", ev.Reason)
	}
	if strings.Contains(ev.Reason, secretValue) || strings.Contains(ev.Reason, "alice") {
		t.Errorf("audit reason leaked argument values: %q", ev.Reason)
	}
}

// fakeMCPHost is an in-memory MCPHost used only by these tests.
type fakeMCPHost struct {
	tools map[string]fakeMCPTool
}

func newFakeMCPHost() *fakeMCPHost {
	return &fakeMCPHost{tools: make(map[string]fakeMCPTool)}
}

type fakeMCPTool struct {
	decl   ToolDeclaration
	result Result
}

func (f *fakeMCPHost) List(_ context.Context) ([]ToolDeclaration, error) {
	out := make([]ToolDeclaration, 0, len(f.tools))
	for _, t := range f.tools {
		out = append(out, t.decl)
	}
	return out, nil
}

func (f *fakeMCPHost) Invoke(_ context.Context, server, tool string, _ map[string]any) (Result, error) {
	t, ok := f.tools[tool]
	if !ok {
		return Result{Status: ResultStatusUnavailable, Reason: "no such tool"}, nil
	}
	if t.decl.ServerName != server {
		return Result{Status: ResultStatusError, Reason: "server mismatch"}, nil
	}
	return t.result, nil
}

type fakeMCPAuditor struct {
	events []AuditEvent
}

func (f *fakeMCPAuditor) Record(ev AuditEvent) {
	f.events = append(f.events, ev)
}

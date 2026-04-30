package tools

import (
	"context"
	"fmt"
)

// MCPHostUnavailableEvidence is the audit/reason marker emitted whenever an
// MCP server reports unavailable through the boundary. Callers and operators
// scan logs and audit feeds for this string to count degraded-mode events.
const MCPHostUnavailableEvidence = "mcp_host_unavailable"

// TrustClass is the trust label attached to a caller (channel, system path,
// or child agent). The MCP host boundary uses these labels to decide whether
// a tool declaration may be exposed or invoked. The string values match the
// progress.json trust_class vocabulary used elsewhere in Gormes (operator,
// gateway, child-agent, system).
type TrustClass string

// Trust class constants. Keep these in sync with the progress.json
// trust_class enum and with subagent.TrustClass; we re-declare the values
// here so the tools package does not depend on subagent.
const (
	TrustClassOperator   TrustClass = "operator"
	TrustClassGateway    TrustClass = "gateway"
	TrustClassChildAgent TrustClass = "child-agent"
	TrustClassSystem     TrustClass = "system"
)

// MCPResultStatus identifies the outcome of one MCP tool invocation through
// the boundary. The status set is intentionally small and audit-friendly.
const (
	MCPResultStatusOK          = "ok"
	MCPResultStatusUnavailable = "unavailable"
	MCPResultStatusError       = "error"
)

// ToolDeclaration is the single source of truth for one MCP/tool host entry.
// One declaration renders both the provider JSON-schema view (for OpenAI-style
// tool descriptors) and the MCP metadata view, so renderers cannot drift.
type ToolDeclaration struct {
	ServerName  string
	ToolName    string
	Description string
	// InputSchema is a JSON-Schema fragment shared by the provider descriptor
	// and the MCP metadata. The boundary never mutates it; renderers return
	// defensive copies.
	InputSchema map[string]any
	// Channels lists allowed channel names; empty means "all channels".
	Channels []string
	// TrustClass lists allowed trust classes; empty means "all trust classes".
	TrustClass []TrustClass
	// Toolset lists the configured toolset names this tool belongs to;
	// empty means "always available, regardless of configured toolset".
	Toolset []string
}

// MCPToolMetadata is the MCP-side view of a ToolDeclaration. It carries the
// minimum metadata an MCP host needs to advertise the tool without leaking
// provider-specific schema wrappers.
type MCPToolMetadata struct {
	ServerName  string
	Name        string
	Description string
	InputSchema map[string]any
}

// ProviderSchema renders the declaration as a provider-side JSON-schema
// fragment shaped like the existing Registry tool descriptor (name,
// description, parameters). Callers receive a defensive copy.
func (d ToolDeclaration) ProviderSchema() map[string]any {
	return map[string]any{
		"name":        d.ToolName,
		"description": d.Description,
		"parameters":  copyJSONMap(d.InputSchema),
	}
}

// MCPMetadata renders the declaration as MCP-side metadata. Callers receive
// a defensive copy of the input schema.
func (d ToolDeclaration) MCPMetadata() MCPToolMetadata {
	return MCPToolMetadata{
		ServerName:  d.ServerName,
		Name:        d.ToolName,
		Description: d.Description,
		InputSchema: copyJSONMap(d.InputSchema),
	}
}

// ToolFilter selects which declarations are exposed to a caller given the
// caller's channel, trust class, and configured toolset list.
type ToolFilter struct {
	Channel    string
	TrustClass TrustClass
	Toolset    []string
}

// Allows reports whether the declaration is visible under this filter. A
// declaration with empty Channels/TrustClass/Toolset is always allowed by
// that dimension. Otherwise the corresponding filter value must match.
func (f ToolFilter) Allows(d ToolDeclaration) bool {
	if len(d.Channels) > 0 && !mcpContainsString(d.Channels, f.Channel) {
		return false
	}
	if len(d.TrustClass) > 0 && !mcpContainsTrustClass(d.TrustClass, f.TrustClass) {
		return false
	}
	if len(d.Toolset) > 0 && !mcpAnyStringOverlap(d.Toolset, f.Toolset) {
		return false
	}
	return true
}

// MCPResult is the audit-friendly outcome of one MCP invocation through the
// boundary. Body carries the tool's response payload; Reason is a
// human-readable, secret-redacted string that is safe to log.
type MCPResult struct {
	Status string
	Body   []byte
	Reason string
}

// MCPAuditEvent is the structured event the boundary emits for every
// invocation. Argument values are never recorded here; the boundary captures
// only the redaction status and the high-level outcome.
type MCPAuditEvent struct {
	Server       string
	Tool         string
	ArgsRedacted bool
	Status       string
	Reason       string
	Unavailable  bool
}

// MCPAuditor records one audit event per invocation.
type MCPAuditor interface {
	Record(MCPAuditEvent)
}

// MCPHost is the small Gormes-native MCP/tool host boundary. Implementations
// might wrap a stdio MCP server, an HTTP MCP server, or an in-memory fake
// (for tests). The boundary does not own transport, retries, or framing.
type MCPHost interface {
	List(ctx context.Context) ([]ToolDeclaration, error)
	Invoke(ctx context.Context, server, tool string, args map[string]any) (MCPResult, error)
}

// RunFiltered wraps a host invocation with filter enforcement and audit
// emission. When redactArgs is true, the boundary never logs argument
// values; only the redaction flag is recorded. When the host returns an
// unavailable status, the rendered Reason includes MCPHostUnavailableEvidence
// so operators can scan for degraded-mode counts without seeing argument
// substrings.
func RunFiltered(
	ctx context.Context,
	host MCPHost,
	filter ToolFilter,
	auditor MCPAuditor,
	server, tool string,
	args map[string]any,
	redactArgs bool,
) MCPResult {
	declared := ToolDeclaration{ServerName: server, ToolName: tool}
	if !filter.Allows(declared) {
		// Filter exclusion is recorded as an error result; the boundary
		// never reveals argument values regardless of redactArgs.
		res := MCPResult{
			Status: MCPResultStatusError,
			Reason: fmt.Sprintf("tool %q on server %q not allowed by filter", tool, server),
		}
		recordAudit(auditor, server, tool, redactArgs, res)
		return res
	}

	res, err := host.Invoke(ctx, server, tool, args)
	if err != nil {
		safe := MCPResult{
			Status: MCPResultStatusError,
			Reason: fmt.Sprintf("invoke failed: %s", err.Error()),
		}
		recordAudit(auditor, server, tool, redactArgs, safe)
		return safe
	}

	// Decorate unavailable results with the public evidence marker. Strip
	// any caller-supplied Reason that might risk leaking argument values:
	// callers cannot promise argument-free Reasons through the boundary.
	if res.Status == MCPResultStatusUnavailable {
		res.Reason = fmt.Sprintf("%s: server=%s tool=%s", MCPHostUnavailableEvidence, server, tool)
	}
	recordAudit(auditor, server, tool, redactArgs, res)
	return res
}

func recordAudit(auditor MCPAuditor, server, tool string, redacted bool, res MCPResult) {
	if auditor == nil {
		return
	}
	auditor.Record(MCPAuditEvent{
		Server:       server,
		Tool:         tool,
		ArgsRedacted: redacted,
		Status:       res.Status,
		Reason:       res.Reason,
		Unavailable:  res.Status == MCPResultStatusUnavailable,
	})
}

func copyJSONMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = copyJSONValue(v)
	}
	return out
}

func copyJSONValue(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		return copyJSONMap(typed)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = copyJSONValue(item)
		}
		return out
	default:
		return typed
	}
}

func mcpContainsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func mcpContainsTrustClass(haystack []TrustClass, needle TrustClass) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func mcpAnyStringOverlap(a, b []string) bool {
	for _, x := range a {
		for _, y := range b {
			if x == y {
				return true
			}
		}
	}
	return false
}

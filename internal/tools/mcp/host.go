package mcp

import (
	"context"
	"fmt"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/access"
)

// HostUnavailableEvidence is the audit/reason marker emitted whenever an
// MCP server reports unavailable through the boundary. Callers and operators
// scan logs and audit feeds for this string to count degraded-mode events.
const HostUnavailableEvidence = "mcp_host_unavailable"

// ResultStatus identifies the outcome of one MCP tool invocation through
// the boundary. The status set is intentionally small and audit-friendly.
const (
	ResultStatusOK          = "ok"
	ResultStatusUnavailable = "unavailable"
	ResultStatusError       = "error"
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
	TrustClass []access.TrustClass
	// Toolset lists the configured toolset names this tool belongs to;
	// empty means "always available, regardless of configured toolset".
	Toolset []string
}

// ToolMetadata is the MCP-side view of a ToolDeclaration. It carries the
// minimum metadata an MCP host needs to advertise the tool without leaking
// provider-specific schema wrappers.
type ToolMetadata struct {
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
func (d ToolDeclaration) MCPMetadata() ToolMetadata {
	return ToolMetadata{
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
	TrustClass access.TrustClass
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

// Result is the audit-friendly outcome of one MCP invocation through the
// boundary. Body carries the tool's response payload; Reason is a
// human-readable, secret-redacted string that is safe to log.
type Result struct {
	Status string
	Body   []byte
	Reason string
}

// AuditEvent is the structured event the boundary emits for every
// invocation. Argument values are never recorded here; the boundary captures
// only the redaction status and the high-level outcome.
type AuditEvent struct {
	Server       string
	Tool         string
	ArgsRedacted bool
	Status       string
	Reason       string
	Unavailable  bool
}

// Auditor records one audit event per invocation.
type Auditor interface {
	Record(AuditEvent)
}

// MCPHost is the small Gormes-native MCP/tool host boundary. Implementations
// might wrap a stdio MCP server, an HTTP MCP server, or an in-memory fake
// (for tests). The boundary does not own transport, retries, or framing.
type Host interface {
	List(ctx context.Context) ([]ToolDeclaration, error)
	Invoke(ctx context.Context, server, tool string, args map[string]any) (Result, error)
}

// RunFiltered wraps a host invocation with filter enforcement and audit
// emission. When redactArgs is true, the boundary never logs argument
// values; only the redaction flag is recorded. When the host returns an
// unavailable status, the rendered Reason includes HostUnavailableEvidence
// so operators can scan for degraded-mode counts without seeing argument
// substrings.
func RunFiltered(
	ctx context.Context,
	host Host,
	filter ToolFilter,
	auditor Auditor,
	server, tool string,
	args map[string]any,
	redactArgs bool,
) Result {
	declared := ToolDeclaration{ServerName: server, ToolName: tool}
	if !filter.Allows(declared) {
		// Filter exclusion is recorded as an error result; the boundary
		// never reveals argument values regardless of redactArgs.
		res := Result{
			Status: ResultStatusError,
			Reason: fmt.Sprintf("tool %q on server %q not allowed by filter", tool, server),
		}
		recordAudit(auditor, server, tool, redactArgs, res)
		return res
	}

	res, err := host.Invoke(ctx, server, tool, args)
	if err != nil {
		safe := Result{
			Status: ResultStatusError,
			Reason: fmt.Sprintf("invoke failed: %s", err.Error()),
		}
		recordAudit(auditor, server, tool, redactArgs, safe)
		return safe
	}

	// Decorate unavailable results with the public evidence marker. Strip
	// any caller-supplied Reason that might risk leaking argument values:
	// callers cannot promise argument-free Reasons through the boundary.
	if res.Status == ResultStatusUnavailable {
		res.Reason = fmt.Sprintf("%s: server=%s tool=%s", HostUnavailableEvidence, server, tool)
	}
	recordAudit(auditor, server, tool, redactArgs, res)
	return res
}

func recordAudit(auditor Auditor, server, tool string, redacted bool, res Result) {
	if auditor == nil {
		return
	}
	auditor.Record(AuditEvent{
		Server:       server,
		Tool:         tool,
		ArgsRedacted: redacted,
		Status:       res.Status,
		Reason:       res.Reason,
		Unavailable:  res.Status == ResultStatusUnavailable,
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

func mcpContainsTrustClass(haystack []access.TrustClass, needle access.TrustClass) bool {
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

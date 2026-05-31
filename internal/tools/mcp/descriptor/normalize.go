package descriptor

import (
	"encoding/json"
	"regexp"
	"strings"
)

// SchemaRejectionReasonInputSchemaNotObject is the SchemaRejection.Reason
// emitted when an MCP tool's InputSchema is non-empty but is not a JSON object
// (e.g. a literal `true`, an array, or a scalar).
const SchemaRejectionReasonInputSchemaNotObject = "input_schema_must_be_object"

// SchemaRejectionReasonDuplicateSanitizedName is emitted when two advertised
// tools collapse to the same provider-visible name after sanitization. Keeping
// both would make later dispatch/registration order-dependent, so the first
// candidate wins and later collisions are reported as rejected inventory.
const SchemaRejectionReasonDuplicateSanitizedName = "duplicate_sanitized_name"

// SchemaRejectionReasonEmptySanitizedName is emitted when a tool advertises an
// empty name. Registering it would create an invalid provider-visible tool
// name and make source-name dispatch ambiguous.
const SchemaRejectionReasonEmptySanitizedName = "empty_sanitized_name"

// defaultInputSchema is the JSON-Schema fragment substituted when an MCP tool
// advertises no InputSchema or an explicit JSON `null`. Hermes' upstream
// _normalize_mcp_input_schema applies the same fallback so providers that
// require an `object` schema do not reject the descriptor outright.
var defaultInputSchema = json.RawMessage(`{"type":"object","properties":{}}`)

// nameSanitizer collapses any character outside [A-Za-z0-9_] to '_', matching
// hermes' sanitize_mcp_name_component semantics so dynamic-discovery names stay
// compatible with provider validation rules.
var nameSanitizer = regexp.MustCompile(`[^A-Za-z0-9_]`)

// RawTool is the verbatim tool envelope returned by an MCP server's tools/list
// response. InputSchema is preserved as raw JSON so downstream schema
// normalization can run separately without lossy round-tripping.
type RawTool struct {
	Name        string
	Description string
	InputSchema json.RawMessage
}

// NormalizedTool is the transport-free descriptor produced by NormalizeTools.
// SourceRaw preserves the verbatim MCP envelope so callers can correlate
// sanitized names back to the on-the-wire identifier.
type NormalizedTool struct {
	Name        string
	ServerName  string
	Description string
	InputSchema json.RawMessage
	SourceRaw   RawTool
}

// SchemaRejection records a single RawTool that was dropped during
// normalization and the reason it was dropped.
type SchemaRejection struct {
	ServerName string
	ToolName   string
	Reason     string
}

// NormalizeResult is the aggregate return value of NormalizeTools.
type NormalizeResult struct {
	Tools    []NormalizedTool
	Rejected []SchemaRejection
}

// NormalizeTools converts an MCP server's tools/list response into native
// NormalizedTool descriptors. Tools whose InputSchema is non-empty but not a
// JSON object are dropped into Rejected with reason
// SchemaRejectionReasonInputSchemaNotObject; the remaining tools land in Tools
// with their names sanitized for provider compatibility.
func NormalizeTools(serverName string, raw []RawTool) NormalizeResult {
	out := NormalizeResult{}
	seenNames := map[string]bool{}
	for _, t := range raw {
		tool, rejection, ok := normalizeToolCandidate(serverName, t, seenNames)
		if !ok {
			out.Rejected = append(out.Rejected, rejection)
			continue
		}
		seenNames[tool.Name] = true
		out.Tools = append(out.Tools, tool)
	}
	return out
}

func normalizeToolCandidate(serverName string, t RawTool, seenNames map[string]bool) (NormalizedTool, SchemaRejection, bool) {
	schema, ok := NormalizeInputSchema(t.InputSchema)
	if !ok {
		return NormalizedTool{}, SchemaRejection{
			ServerName: serverName,
			ToolName:   t.Name,
			Reason:     SchemaRejectionReasonInputSchemaNotObject,
		}, false
	}
	name := SanitizeNameComponent(t.Name)
	if name == "" {
		return NormalizedTool{}, SchemaRejection{
			ServerName: serverName,
			ToolName:   t.Name,
			Reason:     SchemaRejectionReasonEmptySanitizedName,
		}, false
	}
	if seenNames[name] {
		return NormalizedTool{}, SchemaRejection{
			ServerName: serverName,
			ToolName:   t.Name,
			Reason:     SchemaRejectionReasonDuplicateSanitizedName,
		}, false
	}
	return NormalizedTool{
		Name:        name,
		ServerName:  serverName,
		Description: t.Description,
		InputSchema: schema,
		SourceRaw:   t,
	}, SchemaRejection{}, true
}

// SanitizeNameComponent mirrors hermes' upstream helper: characters not in
// [A-Za-z0-9_] are replaced with '_'. The function never panics on empty input.
func SanitizeNameComponent(value string) string {
	return nameSanitizer.ReplaceAllString(value, "_")
}

// NormalizeInputSchema returns the validated JSON-Schema bytes plus a bool
// indicating whether the input was acceptable. Empty / null is replaced with a
// permissive empty object schema; a JSON object is passed through; anything
// else (true/false/number/string/array) is rejected.
func NormalizeInputSchema(raw json.RawMessage) (json.RawMessage, bool) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return defaultInputSchema, true
	}
	if !strings.HasPrefix(trimmed, "{") {
		return nil, false
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, false
	}
	return raw, true
}

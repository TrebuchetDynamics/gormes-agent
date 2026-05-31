package descriptor

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/jsonvalue"
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
		candidate := normalizeToolCandidate(serverName, t, seenNames)
		if !candidate.Accepted {
			out.Rejected = append(out.Rejected, candidate.Rejection)
			continue
		}
		seenNames[candidate.Tool.Name] = true
		out.Tools = append(out.Tools, candidate.Tool)
	}
	return out
}

type normalizedToolCandidate struct {
	Tool      NormalizedTool
	Rejection SchemaRejection
	Accepted  bool
}

func acceptedCandidate(tool NormalizedTool) normalizedToolCandidate {
	return normalizedToolCandidate{Tool: tool, Accepted: true}
}

func rejectedCandidate(serverName, toolName, reason string) normalizedToolCandidate {
	return normalizedToolCandidate{
		Rejection: SchemaRejection{
			ServerName: serverName,
			ToolName:   toolName,
			Reason:     reason,
		},
	}
}

func normalizeToolCandidate(serverName string, t RawTool, seenNames map[string]bool) normalizedToolCandidate {
	schema, ok := NormalizeInputSchema(t.InputSchema)
	if !ok {
		return rejectedCandidate(serverName, t.Name, SchemaRejectionReasonInputSchemaNotObject)
	}
	name := SanitizeNameComponent(t.Name)
	if name == "" {
		return rejectedCandidate(serverName, t.Name, SchemaRejectionReasonEmptySanitizedName)
	}
	if seenNames[name] {
		return rejectedCandidate(serverName, t.Name, SchemaRejectionReasonDuplicateSanitizedName)
	}
	sourceRaw := t
	sourceRaw.InputSchema = jsonvalue.CloneRaw(t.InputSchema)
	return acceptedCandidate(NormalizedTool{
		Name:        name,
		ServerName:  serverName,
		Description: t.Description,
		InputSchema: schema,
		SourceRaw:   sourceRaw,
	})
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
	if jsonvalue.NullishRaw(raw) {
		return jsonvalue.CloneRaw(defaultInputSchema), true
	}
	if !strings.HasPrefix(trimmed, "{") {
		return nil, false
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, false
	}
	return jsonvalue.CloneRaw(raw), true
}

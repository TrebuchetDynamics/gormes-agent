package mcp

import (
	"encoding/json"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/content"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/descriptor"
	mcpstderr "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/stderr"
)

const (
	SchemaRejectionReasonInputSchemaNotObject   = descriptor.SchemaRejectionReasonInputSchemaNotObject
	SchemaRejectionReasonDuplicateSanitizedName = descriptor.SchemaRejectionReasonDuplicateSanitizedName
	SchemaRejectionReasonEmptySanitizedName     = descriptor.SchemaRejectionReasonEmptySanitizedName
)

type NormalizedTool = descriptor.NormalizedTool

type SchemaRejection = descriptor.SchemaRejection

type NormalizeResult = descriptor.NormalizeResult

func NormalizeTools(serverName string, raw []RawTool) NormalizeResult {
	return descriptor.NormalizeTools(serverName, raw)
}

func sanitizeMCPNameComponent(value string) string {
	return descriptor.SanitizeNameComponent(value)
}

func normalizeInputSchema(raw json.RawMessage) (json.RawMessage, bool) {
	return descriptor.NormalizeInputSchema(raw)
}

type StructuredContent = content.Structured
type RenderOptions = content.RenderOptions

func RenderToolCallResult(parts []StructuredContent) string {
	return content.Render(parts)
}

// RenderCallResult renders one parsed tools/call result with server-aware
// resource-link guidance. MCP error envelopes retain their model-facing text;
// only an error with no renderable content falls back to the stable generic
// message used by Hermes.
func RenderCallResult(result CallResult, serverName string) string {
	return RenderCallResultWithOptions(result, RenderOptions{ServerName: serverName})
}

func RenderCallResultWithOptions(result CallResult, opts RenderOptions) string {
	rendered := content.RenderWithOptions(result.Content, opts)
	if rendered == "" && result.IsError {
		return "MCP tool returned an error"
	}
	return rendered
}

type StderrSink = mcpstderr.Sink

func NewBoundedStderrSink(path string, tailBytes int) StderrSink {
	return mcpstderr.NewBoundedSink(path, tailBytes)
}

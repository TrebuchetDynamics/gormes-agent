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

func RenderToolCallResult(parts []StructuredContent) string {
	return content.Render(parts)
}

type StderrSink = mcpstderr.Sink

func NewBoundedStderrSink(path string, tailBytes int) StderrSink {
	return mcpstderr.NewBoundedSink(path, tailBytes)
}

package tools

import mcptools "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp"

const SchemaRejectionReasonInputSchemaNotObject = mcptools.SchemaRejectionReasonInputSchemaNotObject

type NormalizedTool = mcptools.NormalizedTool
type SchemaRejection = mcptools.SchemaRejection
type NormalizeResult = mcptools.NormalizeResult
type StructuredContent = mcptools.StructuredContent
type StderrSink = mcptools.StderrSink

func NormalizeTools(serverName string, raw []MCPRawTool) NormalizeResult {
	return mcptools.NormalizeTools(serverName, raw)
}

func RenderToolCallResult(parts []StructuredContent) string {
	return mcptools.RenderToolCallResult(parts)
}

func NewBoundedStderrSink(path string, tail int) StderrSink {
	return mcptools.NewBoundedStderrSink(path, tail)
}

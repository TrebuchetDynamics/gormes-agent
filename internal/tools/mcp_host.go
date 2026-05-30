package tools

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp"
)

const MCPHostUnavailableEvidence = mcp.HostUnavailableEvidence

const (
	MCPResultStatusOK          = mcp.ResultStatusOK
	MCPResultStatusUnavailable = mcp.ResultStatusUnavailable
	MCPResultStatusError       = mcp.ResultStatusError
)

type ToolDeclaration = mcp.ToolDeclaration
type MCPToolMetadata = mcp.ToolMetadata
type ToolFilter = mcp.ToolFilter
type MCPResult = mcp.Result
type MCPAuditEvent = mcp.AuditEvent
type MCPAuditor = mcp.Auditor
type MCPHost = mcp.Host

func RunFiltered(
	ctx context.Context,
	host MCPHost,
	filter ToolFilter,
	auditor MCPAuditor,
	server, tool string,
	args map[string]any,
	redactArgs bool,
) MCPResult {
	return mcp.RunFiltered(ctx, host, filter, auditor, server, tool, args, redactArgs)
}

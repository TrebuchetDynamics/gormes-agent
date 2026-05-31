package mcp

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/boundary"
)

const HostUnavailableEvidence = boundary.HostUnavailableEvidence

const (
	ResultStatusOK          = boundary.ResultStatusOK
	ResultStatusUnavailable = boundary.ResultStatusUnavailable
	ResultStatusError       = boundary.ResultStatusError
)

type ToolDeclaration = boundary.ToolDeclaration

type ToolMetadata = boundary.ToolMetadata

type ToolFilter = boundary.ToolFilter

type Result = boundary.Result

type AuditEvent = boundary.AuditEvent

type Auditor = boundary.Auditor

type Host = boundary.Host

func RunFiltered(
	ctx context.Context,
	host Host,
	filter ToolFilter,
	auditor Auditor,
	server, tool string,
	args map[string]any,
	redactArgs bool,
) Result {
	return boundary.RunFiltered(ctx, host, filter, auditor, server, tool, args, redactArgs)
}

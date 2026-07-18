package remote

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/callresult"
	mcpconfig "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/config"
	mcpprobe "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/probe"
)

// Session is the smallest shared contract needed by both read-only discovery
// and live MCP tool invocation. Implementations own protocol and transport
// details; callers must close each session.
type Session interface {
	mcpprobe.Session
	CallTool(context.Context, string, map[string]any) (callresult.Result, error)
}

// Connector opens one initialized remote MCP session.
type Connector func(context.Context, mcpconfig.MCPServerDefinition) (Session, error)

// ProbeConnector narrows a live connector to the discovery-only probe seam.
func ProbeConnector(connect Connector) mcpprobe.Connector {
	if connect == nil {
		return nil
	}
	return func(ctx context.Context, def mcpconfig.MCPServerDefinition) (mcpprobe.Session, error) {
		return connect(ctx, def)
	}
}

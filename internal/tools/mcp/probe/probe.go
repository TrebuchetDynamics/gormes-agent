package probe

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/descriptor"
)

type Session interface {
	ListTools(context.Context) ([]descriptor.RawTool, error)
	Close() error
}

type Connector func(context.Context, config.MCPServerDefinition) (Session, error)

func ServerTools(ctx context.Context, servers []config.MCPServerDefinition, connect Connector) map[string][]descriptor.RawTool {
	out := map[string][]descriptor.RawTool{}
	if connect == nil {
		return out
	}
	for _, server := range servers {
		if !server.Enabled || server.Name == "" {
			continue
		}
		session, err := connect(ctx, server)
		if err != nil || session == nil {
			continue
		}
		tools, listErr := session.ListTools(ctx)
		_ = session.Close()
		if listErr != nil {
			continue
		}
		out[server.Name] = append([]descriptor.RawTool(nil), tools...)
	}
	return out
}

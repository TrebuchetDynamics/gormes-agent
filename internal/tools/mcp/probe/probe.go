package probe

import (
	"context"
	"errors"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/descriptor"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/jsonvalue"
)

type Session interface {
	ListTools(context.Context) ([]descriptor.RawTool, error)
	Close() error
}

type Connector func(context.Context, config.MCPServerDefinition) (Session, error)

var ErrConnectorUnavailable = errors.New("mcp probe connector unavailable")

// One connects to exactly one validated server, lists its tools, and closes the
// temporary session. It preserves errors for operator-facing classification;
// the older bulk ServerTools helper intentionally keeps its lossy best-effort
// semantics.
func One(ctx context.Context, server config.MCPServerDefinition, connect Connector) (tools []descriptor.RawTool, err error) {
	if connect == nil {
		return nil, ErrConnectorUnavailable
	}
	session, err := connect(ctx, cloneServerDefinition(server))
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrConnectorUnavailable
	}
	defer func() {
		if closeErr := session.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()
	tools, err = session.ListTools(ctx)
	if err != nil {
		return nil, err
	}
	return cloneRawTools(tools), nil
}

func ServerTools(ctx context.Context, servers []config.MCPServerDefinition, connect Connector) map[string][]descriptor.RawTool {
	out := map[string][]descriptor.RawTool{}
	if connect == nil {
		return out
	}
	seen := map[string]bool{}
	for _, server := range servers {
		if !server.Enabled || server.Name == "" || seen[server.Name] {
			continue
		}
		seen[server.Name] = true
		session, err := connect(ctx, cloneServerDefinition(server))
		if err != nil || session == nil {
			continue
		}
		tools, listErr := session.ListTools(ctx)
		_ = session.Close()
		if listErr != nil {
			continue
		}
		out[server.Name] = cloneRawTools(tools)
	}
	return out
}

func cloneServerDefinition(in config.MCPServerDefinition) config.MCPServerDefinition {
	out := in
	out.Args = append([]string(nil), in.Args...)
	out.Env = cloneStringMap(in.Env)
	out.Headers = cloneStringMap(in.Headers)
	out.Sampling.AllowedModels = append([]string(nil), in.Sampling.AllowedModels...)
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneRawTools(in []descriptor.RawTool) []descriptor.RawTool {
	out := make([]descriptor.RawTool, len(in))
	for i, tool := range in {
		out[i] = tool
		out[i].InputSchema = jsonvalue.CloneRaw(tool.InputSchema)
	}
	return out
}

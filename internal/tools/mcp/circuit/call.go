package circuit

import (
	"context"
	"errors"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/callresult"
)

// ToolCallFunc is the circuit-protected operation invoked against one MCP server.
type ToolCallFunc func(context.Context) (callresult.Result, error)

// CallWithBreaker applies breaker policy around one MCP tool call and records
// server-unreachable evidence for transport failures or MCP isError results.
func CallWithBreaker(ctx context.Context, breaker *Breaker, server string, call ToolCallFunc) (callresult.Result, Evidence, error) {
	if call == nil {
		err := errors.New("mcp call: nil tool call")
		return callresult.Result{}, EvidenceServerUnreachable, err
	}
	if allow, evidence := breaker.BeforeCall(server); !allow {
		return callresult.Result{}, evidence, ErrBreakerOpen
	}
	result, err := call(ctx)
	if err != nil {
		return result, breaker.RecordFailure(server, err), err
	}
	if result.IsError {
		return result, breaker.RecordFailure(server, errors.New("mcp tool reported isError")), nil
	}
	return result, breaker.RecordSuccess(server), nil
}

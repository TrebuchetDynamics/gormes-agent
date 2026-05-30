package gateway

import (
	"context"

	gatewayrpc "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/rpcmode"
)

// RPCRecord is one JSONL object on the Gormes-owned stdio RPC stream.
type RPCRecord = gatewayrpc.RPCRecord

// RPCPromptRequest is the prompt subset accepted by the first Gormes RPC slice.
type RPCPromptRequest = gatewayrpc.RPCPromptRequest

// RPCQueueState mirrors Pi's queue_update shape while remaining Gormes-owned.
type RPCQueueState = gatewayrpc.RPCQueueState

// RPCRuntime is the small runtime seam behind the stdio JSONL loop. Tests can
// provide a fake runtime; production binds this to the local kernel from
// cmd/gormes without starting an HTTP listener or a Pi subprocess.
type RPCRuntime = gatewayrpc.RPCRuntime

type RPCModeOptions = gatewayrpc.RPCModeOptions

// RunRPCMode runs a strict LF-framed stdin/stdout JSONL protocol. Stderr is not
// touched; malformed input and unsupported commands are reported as structured
// response records on stdout and do not terminate the loop.
func RunRPCMode(ctx context.Context, opts RPCModeOptions) error {
	return gatewayrpc.RunRPCMode(ctx, opts)
}

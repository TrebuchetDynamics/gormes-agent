package adaptertest

import (
	"context"
	"sync"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

// ApprovalRecorder records gateway approval resolutions for adapter tests.
// It satisfies gateway.ApprovalResolver-compatible test seams used by channel
// approval button flows.
type ApprovalRecorder struct {
	mu    sync.Mutex
	calls []gateway.ApprovalResolution
	Err   error
}

// ResolveGatewayApproval records res and returns Err.
func (r *ApprovalRecorder) ResolveGatewayApproval(_ context.Context, res gateway.ApprovalResolution) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, res)
	return r.Err
}

// Snapshot returns a copy of recorded approval resolutions.
func (r *ApprovalRecorder) Snapshot() []gateway.ApprovalResolution {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]gateway.ApprovalResolution, len(r.calls))
	copy(out, r.calls)
	return out
}

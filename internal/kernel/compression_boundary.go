package kernel

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

// NotifyCompressionBoundary calls the ContextEngine boundary callback when
// a compression result is accepted into transcript lineage. Skips silently
// when ContextEngine is nil.
func (k *Kernel) NotifyCompressionBoundary(ctx context.Context, reason string) error {
	if k.cfg.ContextEngine == nil {
		return nil
	}
	return k.cfg.ContextEngine.OnCompressionBoundary(ctx, llm.CompressionBoundary{
		OldSessionID: k.sessionID,
		NewSessionID: k.sessionID,
		Reason:       reason,
	})
}

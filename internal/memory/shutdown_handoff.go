package memory

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/memory/lifecycle"
)

type ShutdownMessage = lifecycle.ShutdownMessage
type ShutdownMemoryProvider = lifecycle.ShutdownMemoryProvider
type ShutdownMemoryStatusCode = lifecycle.ShutdownMemoryStatusCode

const (
	ShutdownMemoryInvoked     = lifecycle.ShutdownMemoryInvoked
	ShutdownMemorySkipped     = lifecycle.ShutdownMemorySkipped
	ShutdownMemoryInterrupted = lifecycle.ShutdownMemoryInterrupted
)

type ShutdownMemoryStatus = lifecycle.ShutdownMemoryStatus
type ShutdownHandoffInput = lifecycle.ShutdownHandoffInput

func PerformShutdownHandoff(ctx context.Context, provider ShutdownMemoryProvider, input ShutdownHandoffInput) (ShutdownMemoryStatus, error) {
	return lifecycle.PerformShutdownHandoff(ctx, provider, input)
}

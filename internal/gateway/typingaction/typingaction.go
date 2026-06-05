package typingaction

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

const (
	Name           = "typing"
	ThrottleWindow = 4 * time.Second
	FailedCode     = "typing_action_failed"
)

// ShouldSendForPhase reports whether a render phase should trigger a one-shot
// channel typing action.
func ShouldSendForPhase(phase kernel.Phase) bool {
	switch phase {
	case kernel.PhaseConnecting, kernel.PhaseStreaming, kernel.PhaseReconnecting, kernel.PhaseFinalizing:
		return true
	default:
		return false
	}
}

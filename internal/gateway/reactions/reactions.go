package reactions

import "github.com/TrebuchetDynamics/gormes-agent/internal/kernel"

// ProcessingOutcome is the channel-neutral terminal state for best-effort
// processing reactions.
type ProcessingOutcome string

const (
	ProcessingOutcomeSuccess   ProcessingOutcome = "success"
	ProcessingOutcomeFailure   ProcessingOutcome = "failure"
	ProcessingOutcomeCancelled ProcessingOutcome = "cancelled"
)

// OutcomeForFrame maps a terminal render frame and cancellation state to the
// reaction outcome reported to capable channels.
func OutcomeForFrame(phase kernel.Phase, cancelled bool) ProcessingOutcome {
	if cancelled || phase == kernel.PhaseCancelling {
		return ProcessingOutcomeCancelled
	}
	if phase == kernel.PhaseFailed {
		return ProcessingOutcomeFailure
	}
	return ProcessingOutcomeSuccess
}

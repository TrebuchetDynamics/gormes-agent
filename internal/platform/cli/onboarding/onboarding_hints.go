package onboarding

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/onboarding/hints"

const (
	// BusyInputPromptFlag is the stable onboarding.seen key for the one-time
	// busy-input behavior hint.
	BusyInputPromptFlag = hints.BusyInputPromptFlag
	// ToolProgressPromptFlag is the stable onboarding.seen key for the
	// one-time tool-progress behavior hint.
	ToolProgressPromptFlag = hints.ToolProgressPromptFlag
)

// BusyInputHint returns the one-time operator hint shown after the first input
// received while a turn is already running. It is pure text: callers own seen
// state, persistence, transport, and active-turn policy.
func BusyInputHint(surface, mode string) string {
	return hints.BusyInputHint(surface, mode)
}

// ToolProgressHint returns the one-time operator hint shown when long-running
// tool progress is first surfaced. It does not inspect or mutate display mode.
func ToolProgressHint(surface string) string {
	return hints.ToolProgressHint(surface)
}

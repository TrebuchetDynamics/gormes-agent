package onboarding

import "strings"

const (
	// BusyInputPromptFlag is the stable onboarding.seen key for the one-time
	// busy-input behavior hint.
	BusyInputPromptFlag = "busy_input_prompt"
	// ToolProgressPromptFlag is the stable onboarding.seen key for the
	// one-time tool-progress behavior hint.
	ToolProgressPromptFlag = "tool_progress_prompt"
)

// BusyInputHint returns the one-time operator hint shown after the first input
// received while a turn is already running. It is pure text: callers own seen
// state, persistence, transport, and active-turn policy.
func BusyInputHint(surface, mode string) string {
	surface = normalizeOnboardingSurface(surface)
	switch normalizeBusyInputMode(mode) {
	case "queue":
		if surface == "gateway" {
			return "First-time tip: I queued your message instead of interrupting. Send /busy interrupt to make new messages stop the current task immediately, or /busy status to check. This tip only shows once."
		}
		return "(tip) Your message was queued for the next turn. Use /busy interrupt to make new input stop the current run instead, or /busy steer to inject mid-run. This tip only shows once."
	case "steer":
		if surface == "gateway" {
			return "First-time tip: I steered your message into the current run; it will arrive after the next tool call instead of interrupting. Send /busy interrupt or /busy queue to change this, or /busy status to check. This tip only shows once."
		}
		return "(tip) Your message was steered into the current run; it arrives after the next tool call. Use /busy interrupt or /busy queue to change this. This tip only shows once."
	default:
		if surface == "gateway" {
			return "First-time tip: I interrupted the current task to handle your message. Send /busy queue to queue follow-ups for after the current task, /busy steer to inject them mid-run, or /busy status to check. This tip only shows once."
		}
		return "(tip) Your message interrupted the current run. Use /busy queue to queue messages for the next turn instead, or /busy steer to inject mid-run. This tip only shows once."
	}
}

// ToolProgressHint returns the one-time operator hint shown when long-running
// tool progress is first surfaced. It does not inspect or mutate display mode.
func ToolProgressHint(surface string) string {
	if normalizeOnboardingSurface(surface) == "gateway" {
		return "First-time tip: long-running tool progress is being shown. Send /verbose to cycle progress modes (all -> new -> off). This tip only shows once."
	}
	return "(tip) Long-running tool progress is being shown. Use /verbose to cycle progress display modes (all -> new -> off). This tip only shows once."
}

func normalizeOnboardingSurface(surface string) string {
	switch strings.ToLower(strings.TrimSpace(surface)) {
	case "gateway", "channel", "telegram", "discord", "slack":
		return "gateway"
	default:
		return "cli"
	}
}

func normalizeBusyInputMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "queue":
		return "queue"
	case "steer":
		return "steer"
	default:
		return "interrupt"
	}
}

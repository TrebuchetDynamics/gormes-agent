package routing

import "strings"

type ReasoningEffort string

const (
	ReasoningEffortNone    ReasoningEffort = "none"
	ReasoningEffortMinimal ReasoningEffort = "minimal"
	ReasoningEffortLow     ReasoningEffort = "low"
	ReasoningEffortMedium  ReasoningEffort = "medium"
	ReasoningEffortHigh    ReasoningEffort = "high"
	ReasoningEffortXHigh   ReasoningEffort = "xhigh"
)

type ReasoningEffortSource string

const (
	ReasoningEffortSourceConfigDefault ReasoningEffortSource = "config_default"
	ReasoningEffortSourceTurnOverride  ReasoningEffortSource = "turn_override"
)

type ReasoningEffortState string

const (
	ReasoningEffortStateDefault     ReasoningEffortState = "default"
	ReasoningEffortStateDisabled    ReasoningEffortState = "disabled"
	ReasoningEffortStateOverride    ReasoningEffortState = "override"
	ReasoningEffortStateInvalid     ReasoningEffortState = "invalid"
	ReasoningEffortStateUnsupported ReasoningEffortState = "unsupported"
)

type ReasoningEffortEvidence struct {
	State     ReasoningEffortState
	Source    ReasoningEffortSource
	Requested string
	Effort    ReasoningEffort
	Supported bool
	Forwarded bool
	Reason    string
}

// ProviderStatus is the minimal provider runtime evidence needed to decide
// whether reasoning_effort can be serialized.
type ProviderStatus struct {
	Runtime string
}

func NormalizeReasoningEffort(effort ReasoningEffort) (ReasoningEffort, bool) {
	normalized := ReasoningEffort(strings.ToLower(strings.TrimSpace(string(effort))))
	switch normalized {
	case ReasoningEffortNone,
		ReasoningEffortMinimal,
		ReasoningEffortLow,
		ReasoningEffortMedium,
		ReasoningEffortHigh,
		ReasoningEffortXHigh:
		return normalized, true
	default:
		return "", false
	}
}

func ResolveReasoningEffort(raw string, source ReasoningEffortSource, status ProviderStatus) ReasoningEffortEvidence {
	if source == "" {
		source = ReasoningEffortSourceConfigDefault
	}
	requested := strings.ToLower(strings.TrimSpace(raw))
	supported := ProviderSupportsReasoningEffort(status)
	if requested == "" {
		return ReasoningEffortEvidence{
			State:     ReasoningEffortStateDefault,
			Source:    source,
			Supported: supported,
			Reason:    "no reasoning_effort supplied; provider default applies",
		}
	}

	effort, ok := NormalizeReasoningEffort(ReasoningEffort(requested))
	if !ok {
		return ReasoningEffortEvidence{
			State:     ReasoningEffortStateInvalid,
			Source:    source,
			Requested: requested,
			Supported: supported,
			Reason:    "invalid reasoning_effort " + requested + "; valid values are none, minimal, low, medium, high, xhigh",
		}
	}

	state := ReasoningEffortStateOverride
	reason := "reasoning_effort " + string(effort) + " will be sent on this request"
	if effort == ReasoningEffortNone {
		state = ReasoningEffortStateDisabled
		reason = "reasoning disabled for this request"
	}
	if !supported {
		normalized := normalizeProviderStatus(status)
		return ReasoningEffortEvidence{
			State:     ReasoningEffortStateUnsupported,
			Source:    source,
			Requested: requested,
			Effort:    effort,
			Supported: false,
			Forwarded: false,
			Reason:    "provider runtime " + normalized.Runtime + " does not serialize reasoning_effort",
		}
	}
	return ReasoningEffortEvidence{
		State:     state,
		Source:    source,
		Requested: requested,
		Effort:    effort,
		Supported: true,
		Forwarded: true,
		Reason:    reason,
	}
}

func ProviderSupportsReasoningEffort(status ProviderStatus) bool {
	normalized := normalizeProviderStatus(status)
	return normalized.Runtime == "chat_completions"
}

func normalizeProviderStatus(status ProviderStatus) ProviderStatus {
	if status.Runtime == "" {
		status.Runtime = "unknown"
	}
	return status
}

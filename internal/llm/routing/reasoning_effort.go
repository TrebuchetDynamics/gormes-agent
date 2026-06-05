package routing

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/routing/reasoning"

type ReasoningEffort = reasoning.ReasoningEffort

const (
	ReasoningEffortNone    ReasoningEffort = reasoning.ReasoningEffortNone
	ReasoningEffortMinimal ReasoningEffort = reasoning.ReasoningEffortMinimal
	ReasoningEffortLow     ReasoningEffort = reasoning.ReasoningEffortLow
	ReasoningEffortMedium  ReasoningEffort = reasoning.ReasoningEffortMedium
	ReasoningEffortHigh    ReasoningEffort = reasoning.ReasoningEffortHigh
	ReasoningEffortXHigh   ReasoningEffort = reasoning.ReasoningEffortXHigh
)

type ReasoningEffortSource = reasoning.ReasoningEffortSource

const (
	ReasoningEffortSourceConfigDefault ReasoningEffortSource = reasoning.ReasoningEffortSourceConfigDefault
	ReasoningEffortSourceTurnOverride  ReasoningEffortSource = reasoning.ReasoningEffortSourceTurnOverride
)

type ReasoningEffortState = reasoning.ReasoningEffortState

const (
	ReasoningEffortStateDefault     ReasoningEffortState = reasoning.ReasoningEffortStateDefault
	ReasoningEffortStateDisabled    ReasoningEffortState = reasoning.ReasoningEffortStateDisabled
	ReasoningEffortStateOverride    ReasoningEffortState = reasoning.ReasoningEffortStateOverride
	ReasoningEffortStateInvalid     ReasoningEffortState = reasoning.ReasoningEffortStateInvalid
	ReasoningEffortStateUnsupported ReasoningEffortState = reasoning.ReasoningEffortStateUnsupported
)

type ReasoningEffortEvidence = reasoning.ReasoningEffortEvidence
type ProviderStatus = reasoning.ProviderStatus

func NormalizeReasoningEffort(effort ReasoningEffort) (ReasoningEffort, bool) {
	return reasoning.NormalizeReasoningEffort(effort)
}

func ResolveReasoningEffort(raw string, source ReasoningEffortSource, status ProviderStatus) ReasoningEffortEvidence {
	return reasoning.ResolveReasoningEffort(raw, source, status)
}

func ProviderSupportsReasoningEffort(status ProviderStatus) bool {
	return reasoning.ProviderSupportsReasoningEffort(status)
}

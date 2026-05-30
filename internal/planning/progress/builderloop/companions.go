package builderloop

import "github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/builderloop/scheduler"

type CompanionState = scheduler.CompanionState

type CompanionOptions = scheduler.CompanionOptions

type CompanionDecision = scheduler.CompanionDecision

func CompanionDue(opts CompanionOptions, state CompanionState) CompanionDecision {
	return scheduler.CompanionDue(opts, state)
}

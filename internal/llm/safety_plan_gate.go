package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/safety"

type PlannedCall = safety.PlannedCall
type PlannedActions = safety.PlannedActions
type PlanDecision = safety.PlanDecision
type PlanGate = safety.PlanGate
type Rule = safety.Rule
type DefaultPlanGate = safety.DefaultPlanGate
type TrustClassRule = safety.TrustClassRule

func NewDefaultPlanGate() *DefaultPlanGate {
	return safety.NewDefaultPlanGate()
}

package safety

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/safety/plangate"

// PlannedCall is a single tool invocation the agent intends to execute.
type PlannedCall = plangate.PlannedCall

// PlannedActions is the full set of tool calls the agent plans to execute
// in a single turn.
type PlannedActions = plangate.PlannedActions

// PlanDecision is the result of a plan gate safety check.
type PlanDecision = plangate.Decision

// PlanGate checks planned tool calls before execution. Implementations
// must be safe for concurrent use.
type PlanGate = plangate.Gate

// Rule is a single safety check applied to each planned call.
type Rule = plangate.Rule

// DefaultPlanGate evaluates planned calls against a set of Rules. If any
// rule blocks a call, the entire plan is refused with details about which
// calls were blocked and why.
type DefaultPlanGate = plangate.DefaultGate

// NewDefaultPlanGate creates a plan gate with the TrustClassRule
// pre-registered. Additional rules can be added via AddRule.
func NewDefaultPlanGate() *DefaultPlanGate {
	return plangate.NewDefaultGate()
}

// TrustClassRule blocks calls where the required trust class is not
// present in the caller's allowed roles.
type TrustClassRule = plangate.TrustClassRule

package plangate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// PlannedCall is a single tool invocation the agent intends to execute.
type PlannedCall struct {
	ToolName           string
	Arguments          json.RawMessage
	TrustClassRequired []string
}

// PlannedActions is the full set of tool calls the agent plans to execute
// in a single turn.
type PlannedActions struct {
	Calls []PlannedCall
}

// Decision is the result of a plan gate safety check.
type Decision struct {
	Allowed      bool
	Reason       string
	BlockedCalls []string
}

// Gate checks planned tool calls before execution. Implementations
// must be safe for concurrent use.
type Gate interface {
	CheckPlan(ctx context.Context, plan PlannedActions) (Decision, error)
}

// Rule is a single safety check applied to each planned call.
type Rule interface {
	Check(ctx context.Context, call PlannedCall) (allowed bool, reason string)
}

// DefaultGate evaluates planned calls against a set of Rules. If any
// rule blocks a call, the entire plan is refused with details about which
// calls were blocked and why.
type DefaultGate struct {
	rules []Rule
}

// NewDefaultGate creates a plan gate with the TrustClassRule
// pre-registered. Additional rules can be added via AddRule.
func NewDefaultGate() *DefaultGate {
	return &DefaultGate{
		rules: []Rule{TrustClassRule{}},
	}
}

func (g *DefaultGate) AddRule(r Rule) {
	g.rules = append(g.rules, r)
}

func (g *DefaultGate) CheckPlan(ctx context.Context, plan PlannedActions) (Decision, error) {
	if len(plan.Calls) == 0 {
		return Decision{Allowed: true}, nil
	}

	var blocked []string
	var reasons []string

	for _, call := range plan.Calls {
		for _, rule := range g.rules {
			allowed, reason := rule.Check(ctx, call)
			if !allowed {
				blocked = append(blocked, call.ToolName)
				reasons = append(reasons, fmt.Sprintf("%s: %s", call.ToolName, reason))
				break
			}
		}
	}

	if len(blocked) > 0 {
		return Decision{
			Allowed:      false,
			Reason:       strings.Join(reasons, "; "),
			BlockedCalls: blocked,
		}, nil
	}

	return Decision{Allowed: true}, nil
}

// TrustClassRule blocks calls where the required trust class is not
// present in the caller's allowed roles.
type TrustClassRule struct{}

var allowedTrustClasses = map[string]bool{
	"operator":    true,
	"child-agent": true,
	"system":      true,
}

func (TrustClassRule) Check(ctx context.Context, call PlannedCall) (bool, string) {
	if len(call.TrustClassRequired) == 0 {
		return true, ""
	}

	for _, required := range call.TrustClassRequired {
		if !allowedTrustClasses[required] {
			return false, fmt.Sprintf("requires trust class %q which is not recognized", required)
		}
	}

	return true, ""
}

package react

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/safety/plangate"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/safety/toolgate"
)

type RefuseAction struct {
	ToolName string
	Reason   string
}

type Decision struct {
	Action       string
	ToolCall     *plangate.PlannedCall
	Refusal      *RefuseAction
	ContinueTurn bool
}

type SafetyAdapter struct {
	planGate plangate.Gate
	toolGate toolgate.Gate
}

func NewSafetyAdapter(planGate plangate.Gate, toolGate toolgate.Gate) *SafetyAdapter {
	return &SafetyAdapter{planGate: planGate, toolGate: toolGate}
}

func (a *SafetyAdapter) CheckPlan(ctx context.Context, plan plangate.PlannedActions) (Decision, error) {
	decision, err := a.planGate.CheckPlan(ctx, plan)
	if err != nil {
		return Decision{}, err
	}
	if !decision.Allowed {
		return Decision{
			Action: "refuse",
			Refusal: &RefuseAction{
				ToolName: "plan",
				Reason:   decision.Reason,
			},
			ContinueTurn: true,
		}, nil
	}
	return Decision{Action: "proceed", ContinueTurn: true}, nil
}

func (a *SafetyAdapter) CheckTool(ctx context.Context, call toolgate.CallRequest) (Decision, error) {
	decision, err := a.toolGate.CheckTool(ctx, call)
	if err != nil {
		return Decision{}, err
	}
	if !decision.Allowed {
		return Decision{
			Action: "refuse",
			Refusal: &RefuseAction{
				ToolName: call.ToolName,
				Reason:   decision.Reason,
			},
			ContinueTurn: true,
		}, nil
	}
	return Decision{Action: "execute", ContinueTurn: true}, nil
}

type RefusalEvidence struct {
	ToolName string `json:"tool_name"`
	Reason   string `json:"reason"`
	Action   string `json:"action"`
}

func (r *RefuseAction) MarshalEvidence() (json.RawMessage, error) {
	ev := RefusalEvidence{
		ToolName: r.ToolName,
		Reason:   r.Reason,
		Action:   "refused",
	}
	b, err := json.Marshal(ev)
	if err != nil {
		return nil, fmt.Errorf("marshal refusal evidence: %w", err)
	}
	return b, nil
}

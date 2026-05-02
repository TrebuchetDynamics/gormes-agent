package hermes

import (
	"context"
	"encoding/json"
	"fmt"
)

type RefuseAction struct {
	ToolName string
	Reason   string
}

type ReActDecision struct {
	Action       string
	ToolCall     *PlannedCall
	Refusal      *RefuseAction
	ContinueTurn bool
}

type ReActSafetyAdapter struct {
	planGate PlanGate
	toolGate ToolGate
}

func NewReActSafetyAdapter(planGate PlanGate, toolGate ToolGate) *ReActSafetyAdapter {
	return &ReActSafetyAdapter{planGate: planGate, toolGate: toolGate}
}

func (a *ReActSafetyAdapter) CheckPlan(ctx context.Context, plan PlannedActions) (ReActDecision, error) {
	decision, err := a.planGate.CheckPlan(ctx, plan)
	if err != nil {
		return ReActDecision{}, err
	}
	if !decision.Allowed {
		return ReActDecision{
			Action: "refuse",
			Refusal: &RefuseAction{
				ToolName: "plan",
				Reason:   decision.Reason,
			},
			ContinueTurn: true,
		}, nil
	}
	return ReActDecision{Action: "proceed", ContinueTurn: true}, nil
}

func (a *ReActSafetyAdapter) CheckTool(ctx context.Context, call ToolCallRequest) (ReActDecision, error) {
	decision, err := a.toolGate.CheckTool(ctx, call)
	if err != nil {
		return ReActDecision{}, err
	}
	if !decision.Allowed {
		return ReActDecision{
			Action: "refuse",
			Refusal: &RefuseAction{
				ToolName: call.ToolName,
				Reason:   decision.Reason,
			},
			ContinueTurn: true,
		}, nil
	}
	return ReActDecision{Action: "execute", ContinueTurn: true}, nil
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

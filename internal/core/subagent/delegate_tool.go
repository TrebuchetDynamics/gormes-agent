package subagent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	delegatepolicy "github.com/TrebuchetDynamics/gormes-agent/internal/core/subagent/delegate"
)

// DelegateTool is the Go-native delegate_task tool.
type DelegateTool struct {
	manager SubagentManager
	drafter CandidateDrafter
}

type CandidateDraftRequest struct {
	Slug            string
	Goal            string
	Summary         string
	SourceRunID     string
	ParentSessionID string
	ChildAgentID    string
	ToolNames       []string
}

type CandidateDrafter interface {
	DraftCandidate(ctx context.Context, req CandidateDraftRequest) (string, error)
}

// NewDelegateTool wires a DelegateTool to the supplied SubagentManager.
func NewDelegateTool(m SubagentManager, drafter CandidateDrafter) *DelegateTool {
	return &DelegateTool{manager: m, drafter: drafter}
}

func (*DelegateTool) Name() string { return "delegate_task" }

func (*DelegateTool) Description() string {
	return "Delegate a task to a subagent for parallel execution. The subagent runs with its own context, returns a structured JSON result."
}

// Timeout returns 0 so the executor does not impose a deadline; per-subagent
// timeouts are governed via SubagentConfig.Timeout.
func (*DelegateTool) Timeout() time.Duration { return 0 }

func (*DelegateTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"goal":           {"type": "string", "description": "Task goal for the subagent"},
			"context":        {"type": "string", "description": "Optional additional context"},
			"tasks":          {"description": "Batch tasks as an array of task objects, or a JSON string encoding that array", "oneOf": [{"type": "array", "items": {"type": "object"}}, {"type": "string"}]},
			"max_iterations": {"type": "integer", "description": "Max LLM turns for the subagent"},
			"toolsets":       {"type": "string", "description": "Comma-separated tool names to allowlist for the child run"},
			"draft_candidate_slug": {"type": "string", "description": "Optional inactive skill slug to draft from a successful delegated run"},
			"allow_no_tool_draft": {"type": "boolean", "description": "Explicit override allowing candidate drafting even when the child emitted no tool calls"}
		},
		"anyOf": [{"required": ["goal"]}, {"required": ["tasks"]}]
	}`)
}

type delegateArgs = delegatepolicy.Args
type delegateBatchTask = delegatepolicy.Task

func (t *DelegateTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in delegateArgs
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("delegate_task: invalid args: %w", err)
	}
	if delegatepolicy.HasTasks(in.Tasks) {
		taskList, err := delegatepolicy.ParseTasks(in.Tasks)
		if err != nil {
			return nil, err
		}
		if t.manager == nil {
			return nil, errors.New("delegate_task: manager is required")
		}
		return t.executeBatch(ctx, in, taskList)
	}
	if strings.TrimSpace(in.Goal) == "" {
		return nil, errors.New("delegate_task: goal is required")
	}
	if t.manager == nil {
		return nil, errors.New("delegate_task: manager is required")
	}

	enabled := delegatepolicy.SplitToolsets(in.Toolsets)

	sa, err := t.manager.Spawn(ctx, SubagentConfig{
		Goal:          strings.TrimSpace(in.Goal),
		Context:       strings.TrimSpace(in.Context),
		MaxIterations: in.MaxIterations,
		EnabledTools:  enabled,
	})
	if err != nil {
		return nil, fmt.Errorf("delegate_task: spawn: %w", err)
	}

	result, err := sa.WaitForResult(ctx)
	if err != nil {
		_ = t.manager.Interrupt(sa, "parent ctx cancelled")
		return nil, err
	}

	var candidateID string
	var candidateErr error
	if t.drafter != nil && strings.TrimSpace(in.DraftSlug) != "" && result.Status == StatusCompleted {
		if len(result.ToolCalls) > 0 || in.AllowNoTool {
			candidateID, candidateErr = t.drafter.DraftCandidate(ctx, CandidateDraftRequest{
				Slug:         strings.TrimSpace(in.DraftSlug),
				Goal:         strings.TrimSpace(in.Goal),
				Summary:      result.Summary,
				SourceRunID:  result.ID,
				ChildAgentID: result.ID,
				ToolNames:    toolCallNames(result.ToolCalls),
			})
		}
	}

	out := delegateResultEnvelope(result)
	if candidateID != "" {
		out["candidate_id"] = candidateID
	}
	if candidateErr != nil {
		out["candidate_error"] = candidateErr.Error()
	}
	return json.Marshal(out)
}

func (t *DelegateTool) executeBatch(ctx context.Context, in delegateArgs, tasks []delegateBatchTask) (json.RawMessage, error) {
	if len(tasks) == 0 {
		return nil, errors.New("delegate_task: No tasks provided.")
	}
	if len(tasks) > DefaultMaxConcurrent {
		return nil, fmt.Errorf("delegate_task: too many tasks: %d provided, but max_concurrent_children is %d", len(tasks), DefaultMaxConcurrent)
	}

	cfgs := make([]SubagentConfig, 0, len(tasks))
	for _, task := range tasks {
		toolsets := task.Toolsets
		if strings.TrimSpace(toolsets) == "" {
			toolsets = in.Toolsets
		}
		cfgs = append(cfgs, SubagentConfig{
			Goal:          strings.TrimSpace(task.Goal),
			Context:       strings.TrimSpace(task.Context),
			MaxIterations: delegatepolicy.FirstPositive(task.MaxIterations, in.MaxIterations),
			EnabledTools:  delegatepolicy.SplitToolsets(toolsets),
		})
	}

	start := time.Now()
	results, err := t.manager.SpawnBatch(ctx, cfgs, DefaultMaxConcurrent)
	if err != nil {
		return nil, fmt.Errorf("delegate_task: spawn batch: %w", err)
	}

	out := make([]map[string]any, 0, len(results))
	for _, result := range results {
		out = append(out, delegateResultEnvelope(result))
	}
	return json.Marshal(map[string]any{
		"results":           out,
		"total_duration_ms": time.Since(start).Milliseconds(),
	})
}

func delegateResultEnvelope(result *SubagentResult) map[string]any {
	if result == nil {
		return delegatepolicy.ResultEnvelope(nil)
	}
	return delegatepolicy.ResultEnvelope(&delegatepolicy.Result{
		ID:           result.ID,
		Status:       string(result.Status),
		Summary:      result.Summary,
		ExitReason:   result.ExitReason,
		Duration:     result.Duration,
		Iterations:   result.Iterations,
		Error:        result.Error,
		ToolCalls:    result.ToolCalls,
		HasToolCalls: len(result.ToolCalls) > 0,
	})
}

func toolCallNames(calls []ToolCallInfo) []string {
	out := make([]string, 0, len(calls))
	for _, call := range calls {
		if name := strings.TrimSpace(call.Name); name != "" {
			out = append(out, name)
		}
	}
	return out
}

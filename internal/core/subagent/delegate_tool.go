package subagent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
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

type delegateArgs struct {
	Goal          string          `json:"goal"`
	Context       string          `json:"context"`
	Tasks         json.RawMessage `json:"tasks"`
	MaxIterations int             `json:"max_iterations"`
	Toolsets      string          `json:"toolsets"`
	DraftSlug     string          `json:"draft_candidate_slug"`
	AllowNoTool   bool            `json:"allow_no_tool_draft"`
}

type delegateBatchTask struct {
	Goal          string `json:"goal"`
	Context       string `json:"context"`
	MaxIterations int    `json:"max_iterations"`
	Toolsets      string `json:"toolsets"`
}

func (t *DelegateTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in delegateArgs
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("delegate_task: invalid args: %w", err)
	}
	if hasDelegateTasks(in.Tasks) {
		taskList, err := parseDelegateTasks(in.Tasks)
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

	var enabled []string
	if in.Toolsets != "" {
		for _, s := range strings.Split(in.Toolsets, ",") {
			if s = strings.TrimSpace(s); s != "" {
				enabled = append(enabled, s)
			}
		}
	}

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
			MaxIterations: firstPositive(task.MaxIterations, in.MaxIterations),
			EnabledTools:  splitToolsets(toolsets),
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

func hasDelegateTasks(raw json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(raw))
	return trimmed != "" && trimmed != "null"
}

func parseDelegateTasks(raw json.RawMessage) ([]delegateBatchTask, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, errors.New("delegate_task: Provide either 'goal' (single task) or 'tasks' (batch).")
	}
	if strings.HasPrefix(trimmed, `"`) {
		var encoded string
		if err := json.Unmarshal(raw, &encoded); err != nil {
			return nil, fmt.Errorf("delegate_task: tasks string is invalid JSON: %w", err)
		}
		encoded = strings.TrimSpace(encoded)
		if encoded == "" {
			return nil, errors.New("delegate_task: Provide either 'goal' (single task) or 'tasks' (batch).")
		}
		tasks, err := parseDelegateTaskArray([]byte(encoded), true)
		if err != nil {
			return nil, err
		}
		return tasks, nil
	}
	return parseDelegateTaskArray(raw, false)
}

func parseDelegateTaskArray(raw []byte, fromString bool) ([]delegateBatchTask, error) {
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		if fromString {
			var decoded any
			if json.Unmarshal(raw, &decoded) == nil {
				return nil, fmt.Errorf("delegate_task: tasks must be a JSON array of task objects; parsed %s instead.", jsonTypeName(decoded))
			}
			return nil, fmt.Errorf("delegate_task: tasks must be a JSON array of task objects; received a string that could not be parsed as JSON (%v).", err)
		}
		return nil, fmt.Errorf("delegate_task: tasks must be a JSON array of task objects: %w", err)
	}
	if len(items) == 0 {
		return nil, errors.New("delegate_task: No tasks provided.")
	}

	tasks := make([]delegateBatchTask, 0, len(items))
	for i, item := range items {
		if !strings.HasPrefix(strings.TrimSpace(string(item)), "{") {
			return nil, fmt.Errorf("delegate_task: Task %d must be an object, got %s.", i, rawJSONTypeName(item))
		}
		var task delegateBatchTask
		if err := json.Unmarshal(item, &task); err != nil {
			return nil, fmt.Errorf("delegate_task: Task %d must be an object: %w", i, err)
		}
		if strings.TrimSpace(task.Goal) == "" {
			return nil, fmt.Errorf("delegate_task: Task %d is missing a 'goal'.", i)
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func delegateResultEnvelope(result *SubagentResult) map[string]any {
	if result == nil {
		return map[string]any{
			"status":      string(StatusError),
			"summary":     "",
			"exit_reason": "nil_result",
			"duration_ms": int64(0),
			"iterations":  0,
			"error":       "subagent returned nil result",
		}
	}
	out := map[string]any{
		"id":          result.ID,
		"status":      string(result.Status),
		"summary":     result.Summary,
		"exit_reason": result.ExitReason,
		"duration_ms": result.Duration.Milliseconds(),
		"iterations":  result.Iterations,
		"error":       result.Error,
	}
	if len(result.ToolCalls) > 0 {
		out["tool_calls"] = result.ToolCalls
	}
	return out
}

func splitToolsets(toolsets string) []string {
	if strings.TrimSpace(toolsets) == "" {
		return nil
	}
	var enabled []string
	for _, s := range strings.Split(toolsets, ",") {
		if s = strings.TrimSpace(s); s != "" {
			enabled = append(enabled, s)
		}
	}
	return enabled
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func rawJSONTypeName(raw []byte) string {
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return "invalid"
	}
	return jsonTypeName(decoded)
}

func jsonTypeName(value any) string {
	switch value.(type) {
	case map[string]any:
		return "object"
	case []any:
		return "array"
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "bool"
	case nil:
		return "null"
	default:
		return "unknown"
	}
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

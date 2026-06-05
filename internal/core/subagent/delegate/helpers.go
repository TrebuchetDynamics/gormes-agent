package delegate

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Args is the JSON shape accepted by the delegate_task tool.
type Args struct {
	Goal          string          `json:"goal"`
	Context       string          `json:"context"`
	Tasks         json.RawMessage `json:"tasks"`
	MaxIterations int             `json:"max_iterations"`
	Toolsets      string          `json:"toolsets"`
	DraftSlug     string          `json:"draft_candidate_slug"`
	AllowNoTool   bool            `json:"allow_no_tool_draft"`
}

// Task is one parsed batch task from delegate_task arguments.
type Task struct {
	Goal          string `json:"goal"`
	Context       string `json:"context"`
	MaxIterations int    `json:"max_iterations"`
	Toolsets      string `json:"toolsets"`
}

// Result is the package-neutral result envelope input for delegate_task output.
type Result struct {
	ID           string
	Status       string
	Summary      string
	ExitReason   string
	Duration     time.Duration
	Iterations   int
	Error        string
	ToolCalls    any
	HasToolCalls bool
}

func HasTasks(raw json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(raw))
	return trimmed != "" && trimmed != "null"
}

func ParseTasks(raw json.RawMessage) ([]Task, error) {
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
		tasks, err := parseTaskArray([]byte(encoded), true)
		if err != nil {
			return nil, err
		}
		return tasks, nil
	}
	return parseTaskArray(raw, false)
}

func parseTaskArray(raw []byte, fromString bool) ([]Task, error) {
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

	tasks := make([]Task, 0, len(items))
	for i, item := range items {
		if !strings.HasPrefix(strings.TrimSpace(string(item)), "{") {
			return nil, fmt.Errorf("delegate_task: Task %d must be an object, got %s.", i, rawJSONTypeName(item))
		}
		var task Task
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

func ResultEnvelope(result *Result) map[string]any {
	if result == nil {
		return map[string]any{
			"status":      "error",
			"summary":     "",
			"exit_reason": "nil_result",
			"duration_ms": int64(0),
			"iterations":  0,
			"error":       "subagent returned nil result",
		}
	}
	out := map[string]any{
		"id":          result.ID,
		"status":      result.Status,
		"summary":     result.Summary,
		"exit_reason": result.ExitReason,
		"duration_ms": result.Duration.Milliseconds(),
		"iterations":  result.Iterations,
		"error":       result.Error,
	}
	if result.HasToolCalls {
		out["tool_calls"] = result.ToolCalls
	}
	return out
}

func SplitToolsets(toolsets string) []string {
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

func FirstPositive(values ...int) int {
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

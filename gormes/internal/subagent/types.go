package subagent

import "time"

type Spec struct {
	Goal          string
	Context       string
	Model         string
	AllowedTools  []string
	MaxIterations int
	Timeout       time.Duration
	Depth         int
}

type EventType string

const (
	EventStarted   EventType = "started"
	EventProgress  EventType = "progress"
	EventToolCall  EventType = "tool_call"
	EventCompleted EventType = "completed"
	EventFailed    EventType = "failed"
)

type Event struct {
	Type      EventType `json:"type"`
	Message   string    `json:"message,omitempty"`
	ToolName  string    `json:"tool_name,omitempty"`
	Iteration int       `json:"iteration,omitempty"`
}

type ResultStatus string

const (
	StatusCompleted ResultStatus = "completed"
	StatusFailed    ResultStatus = "failed"
	StatusCancelled ResultStatus = "cancelled"
	StatusTimedOut  ResultStatus = "timed_out"
)

type Result struct {
	RunID        string       `json:"run_id"`
	Status       ResultStatus `json:"status"`
	Summary      string       `json:"summary,omitempty"`
	Error        string       `json:"error,omitempty"`
	FinishReason string       `json:"finish_reason,omitempty"`
	ToolCalls    []string     `json:"tool_calls,omitempty"`
}

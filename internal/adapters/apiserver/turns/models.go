package turns

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm"

// ChatMessage is the normalized text shape passed from HTTP into gateway turns.
type ChatMessage struct {
	Role         string                   `json:"role"`
	Content      string                   `json:"content"`
	ContentParts []llm.MessageContentPart `json:"content_parts,omitempty"`
	ToolCalls    []ToolCall               `json:"tool_calls,omitempty"`
	ToolCallID   string                   `json:"tool_call_id,omitempty"`
	Name         string                   `json:"name,omitempty"`
}

// ToolCall is the OpenAI function-call metadata preserved in response chains.
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Request is the chat-completions request after OpenAI message/content
// normalization and session-handle resolution.
type Request struct {
	Model            string
	UserMessage      string
	UserContentParts []llm.MessageContentPart
	History          []ChatMessage
	SystemPrompt     string
	SessionID        string
}

// Usage is the OpenAI-compatible token accounting shape used by both normal
// and streaming chat-completion responses.
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// Result is the native turn-loop result consumed by HTTP response writers.
type Result struct {
	Content      string
	SessionID    string
	Usage        Usage
	FinishReason string
	Messages     []ChatMessage
}

// StreamCallbacks receives token deltas from a streaming native turn.
type StreamCallbacks struct {
	OnToken        func(string) error
	OnToolProgress func(ToolProgressEvent) error
}

// ToolProgressEvent is the dashboard-facing progress item emitted by native
// run streams for long-running tool activity.
type ToolProgressEvent struct {
	Name    string `json:"name,omitempty"`
	Preview string `json:"preview,omitempty"`
	Status  string `json:"status,omitempty"`
}

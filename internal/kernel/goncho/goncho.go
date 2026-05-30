package goncho

import (
	"context"
	"errors"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

var ErrUnavailable = errors.New("goncho not configured")

type Store interface {
	AppendTurn(ctx context.Context, peer, sessionKey, role, content string) error
	GetContext(ctx context.Context, sessionKey string, maxTokens int) (string, error)
	OnSessionEnd(ctx context.Context, sessionKey string, messages []llm.Message) error
	Observe(ctx context.Context, obs Observation) error
}

type ObservationKind string

const (
	ObservationSessionStart      ObservationKind = "session_start"
	ObservationUserPrompt        ObservationKind = "user_prompt"
	ObservationToolCall          ObservationKind = "tool_call"
	ObservationToolResult        ObservationKind = "tool_result"
	ObservationToolError         ObservationKind = "tool_error"
	ObservationAssistantResponse ObservationKind = "assistant_response"
	ObservationCompact           ObservationKind = "compact"
	ObservationSessionEnd        ObservationKind = "session_end"
	ObservationCustom            ObservationKind = "custom"
)

type Observation struct {
	Kind       ObservationKind
	PeerID     string
	SessionKey string
	ContextID  string
	Input      string
	Output     string
	Success    *bool
	Metadata   map[string]string
	ObservedAt time.Time
	Reason     string
}

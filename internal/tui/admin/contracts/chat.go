package contracts

import (
	"context"

	goncho "github.com/TrebuchetDynamics/goncho/dynamicagents"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

// ChatRequest is the slim admin Chat tab request passed through the responder
// seam. Tests inject a fake responder; production uses the configured Hermes
// provider client and keeps history in memory for this admin session only.
type ChatRequest struct {
	AgentID  string
	Prompt   string
	Messages []llm.Message
}

// ChatResponder handles one submitted chat turn.
type ChatResponder interface {
	Respond(context.Context, ChatRequest) (string, error)
}

// AgentLister returns runtime-spawned agents available for the Chat tab's
// active-agent picker.
type AgentLister interface {
	ListAgents(context.Context) ([]goncho.AgentRecord, error)
}

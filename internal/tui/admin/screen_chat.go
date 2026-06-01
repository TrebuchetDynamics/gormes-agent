package admin

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/admin/chat"
	chatcontracts "github.com/TrebuchetDynamics/gormes-agent/internal/tui/admin/contracts/chat"
)

// ChatRequest is the slim admin Chat tab request passed through the responder
// seam. Root aliases preserve the original admin package API while the focused
// contracts package lets responders/listers avoid depending on the concrete
// chat screen implementation.
type ChatRequest = chatcontracts.Request

// ChatResponder handles one submitted chat turn.
type ChatResponder = chatcontracts.Responder

// AgentLister returns runtime-spawned agents available for the Chat tab's
// active-agent picker.
type AgentLister = chatcontracts.AgentLister

// ChatScreen is a slim in-admin chat surface.
type ChatScreen = chat.Screen

// ChatOption configures a ChatScreen.
type ChatOption = chat.Option

type chatResponderFunc func(context.Context, ChatRequest) (string, error)

func (f chatResponderFunc) Respond(ctx context.Context, req ChatRequest) (string, error) {
	return f(ctx, req)
}

// WithChatResponder replaces the default provider-backed responder.
func WithChatResponder(responder ChatResponder) ChatOption {
	return chat.WithChatResponder(responder)
}

// WithAgentLister replaces the default Goncho dynamic-agent lister.
func WithAgentLister(lister AgentLister) ChatOption {
	return chat.WithAgentLister(lister)
}

// NewChatScreen returns the admin Chat tab.
func NewChatScreen(opts ...ChatOption) *ChatScreen {
	return chat.NewScreen(opts...)
}

package gateway

import (
	"context"
	"strings"

	gatewaypersonality "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/personality"
)

// handlePersonalityCommand handles /personality subcommands (list, switch, none).
func (m *Manager) handlePersonalityCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	personalities := m.loadPersonalities()
	arg := gatewaypersonality.ParseArg(ev.Text)

	// /personality — list available personalities
	if arg == "" {
		_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID,
			gatewaypersonality.RenderList(m.activePersonality(), personalities, 60))
		return
	}

	// /personality none — clear
	if strings.ToLower(strings.TrimSpace(arg)) == "none" {
		m.setActivePersonality("")
		_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, "Personality cleared.")
		return
	}

	// /personality <name> — switch
	name := strings.ToLower(strings.TrimSpace(arg))
	prompt, ok := personalities[name]
	if !ok {
		_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID,
			gatewaypersonality.RenderUnknown(name, personalities))
		return
	}
	m.setActivePersonality(name)
	_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID,
		gatewaypersonality.RenderSetConfirmation(name))
	_ = prompt // used at prompt assembly time
}

// loadPersonalities returns the configured personality map. Defaults to nil.
func (m *Manager) loadPersonalities() map[string]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.personalityPrompts
}

// activePersonality returns the currently active personality name.
func (m *Manager) activePersonality() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.activePersonalityName
}

// setActivePersonality sets the active personality name.
func (m *Manager) setActivePersonality(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activePersonalityName = name
}

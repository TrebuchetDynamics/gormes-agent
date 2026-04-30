package gateway

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func (m *Manager) updateTypingAction(ctx context.Context, ch Channel, chatID string, phase kernel.Phase) {
	if m == nil || ch == nil || chatID == "" {
		return
	}
	if isTypingActionPhase(phase) {
		m.startTypingAction(ctx, ch, chatID)
		return
	}
	if isTypingActionTerminalPhase(phase) {
		m.stopTypingAction()
	}
}

func isTypingActionPhase(phase kernel.Phase) bool {
	switch phase {
	case kernel.PhaseConnecting, kernel.PhaseStreaming, kernel.PhaseReconnecting, kernel.PhaseFinalizing:
		return true
	default:
		return false
	}
}

func isTypingActionTerminalPhase(phase kernel.Phase) bool {
	switch phase {
	case kernel.PhaseIdle, kernel.PhaseFailed, kernel.PhaseCancelling:
		return true
	default:
		return false
	}
}

func (m *Manager) startTypingAction(ctx context.Context, ch Channel, chatID string) {
	key := ch.Name() + ":" + chatID
	m.turnMu.Lock()
	if m.typingStop != nil && m.typingKey == key {
		m.turnMu.Unlock()
		return
	}
	if stop := m.typingStop; stop != nil {
		m.typingStop = nil
		m.typingKey = ""
		m.turnMu.Unlock()
		stop()
	} else {
		m.turnMu.Unlock()
	}

	typing, ok := ch.(TypingCapable)
	if !ok {
		return
	}
	stop, err := typing.StartTyping(ctx, chatID)
	if err != nil || stop == nil {
		return
	}
	m.turnMu.Lock()
	m.typingStop = stop
	m.typingKey = key
	m.turnMu.Unlock()
}

func (m *Manager) stopTypingAction() {
	m.turnMu.Lock()
	stop := m.typingStop
	m.typingStop = nil
	m.typingKey = ""
	m.turnMu.Unlock()
	if stop != nil {
		stop()
	}
}

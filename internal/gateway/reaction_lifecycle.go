package gateway

import (
	"context"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/reactions"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func (m *Manager) startProcessingReaction(ctx context.Context, ch Channel, ev InboundEvent) {
	reactions, ok := ch.(ReactionCapable)
	if !ok {
		return
	}
	chatID := strings.TrimSpace(ev.ChatID)
	msgID := strings.TrimSpace(ev.MsgID)
	if chatID == "" || msgID == "" {
		return
	}
	if err := reactions.OnProcessingStart(ctx, chatID, msgID); err != nil {
		m.log.Debug("reaction_unavailable", "platform", ch.Name(), "stage", "start", "err", err)
	}
}

func (m *Manager) completeProcessingReaction(ctx context.Context, ch Channel, outcome ProcessingOutcome) {
	reactions, ok := ch.(ReactionCapable)
	if !ok {
		return
	}
	state, ok := m.activeTurnSnapshot()
	if !ok {
		return
	}
	chatID := strings.TrimSpace(state.ChatID)
	msgID := strings.TrimSpace(state.MsgID)
	if chatID == "" || msgID == "" {
		return
	}
	if err := reactions.OnProcessingComplete(ctx, chatID, msgID, outcome); err != nil {
		m.log.Debug("reaction_unavailable", "platform", ch.Name(), "stage", "complete", "outcome", outcome, "err", err)
	}
}

func processingOutcomeForFrame(phase kernel.Phase, cancelled bool) ProcessingOutcome {
	return reactions.OutcomeForFrame(phase, cancelled)
}

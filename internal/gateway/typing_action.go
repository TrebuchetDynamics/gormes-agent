package gateway

import (
	"context"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/typingaction"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

const (
	typingActionName           = typingaction.Name
	typingActionThrottleWindow = typingaction.ThrottleWindow
	typingActionFailedCode     = typingaction.FailedCode
)

// TypingActionEvidence carries redacted non-fatal typing-action failure
// evidence. It deliberately omits raw API errors, chat IDs, and credentials.
type TypingActionEvidence struct {
	Code    string
	Message string
}

// TypingActionEvidenceSink receives redacted typing-action failure evidence.
type TypingActionEvidenceSink func(TypingActionEvidence)

func isTypingActionPhase(phase kernel.Phase) bool {
	return typingaction.ShouldSendForPhase(phase)
}

func (m *Manager) maybeSendTypingAction(ctx context.Context, ch Channel, phase kernel.Phase, chatID, threadID string) {
	if !isTypingActionPhase(phase) {
		return
	}
	actor, ok := ch.(TypingActionCapable)
	if !ok {
		return
	}
	if telegramDMTopicReplyFallbackLane(ch.Name(), chatID, threadID) {
		return
	}

	key := ch.Name() + "\x00" + chatID + "\x00" + threadID
	now := m.now()
	m.typingActionMu.Lock()
	if m.typingActionLast == nil {
		m.typingActionLast = map[string]time.Time{}
	}
	last := m.typingActionLast[key]
	if !last.IsZero() && now.Sub(last) < typingActionThrottleWindow {
		m.typingActionMu.Unlock()
		return
	}
	m.typingActionLast[key] = now
	m.typingActionMu.Unlock()

	var err error
	if threadID != "" {
		if threadActor, ok := ch.(ThreadTypingActionCapable); ok {
			err = threadActor.SendThreadChatAction(ctx, chatID, threadID, typingActionName)
		} else {
			err = actor.SendChatAction(ctx, chatID, typingActionName)
		}
	} else {
		err = actor.SendChatAction(ctx, chatID, typingActionName)
	}
	if err != nil {
		m.recordTypingActionFailure(ch.Name())
	}
}

func (m *Manager) recordTypingActionFailure(platform string) {
	ev := TypingActionEvidence{
		Code:    typingActionFailedCode,
		Message: "typing action failed",
	}
	m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
		Platform:      platform,
		PlatformState: PlatformStateRunning,
		ErrorMessage:  ev.Code,
	})
	if m.cfg.TypingActionEvidenceSink == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			m.log.Error("typing_action_evidence_sink_panic", "panic", r)
		}
	}()
	m.cfg.TypingActionEvidenceSink(ev)
}

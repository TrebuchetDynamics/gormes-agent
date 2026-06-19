package gateway

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

type gatewayCommandHandler func(*Manager, context.Context, Channel, InboundEvent) error

func resetCommandErrorText(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return ""
	}
	lower := strings.ToLower(msg)
	compact := compactResetSecretSeparators(lower)
	for _, marker := range []string{"token", "api_key", "apikey", "authorization", "bearer", "secret", "password"} {
		if strings.Contains(lower, marker) || strings.Contains(compact, marker) {
			return "[redacted]"
		}
	}
	replacer := strings.NewReplacer("`", "'", "*", "'", "#", "＃")
	return strings.Join(strings.Fields(replacer.Replace(msg)), " ")
}

func compactResetSecretSeparators(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

var gatewayCommandHandlers = map[EventKind]gatewayCommandHandler{
	EventStart: func(m *Manager, ctx context.Context, ch Channel, ev InboundEvent) error {
		if _, err := m.sendWithHooks(ctx, ch, ev.ChatID, startGreeting); err != nil {
			m.log.Warn("send greeting", "platform", ev.Platform, "chat_id", ev.ChatID, "err", err)
		}
		return nil
	},
	EventCancel: func(m *Manager, _ context.Context, _ Channel, _ InboundEvent) error {
		m.markTurnCancelled()
		if k := m.activeTurnKernel(); k != nil {
			_ = k.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventCancel})
		}
		return nil
	},
	EventReset: func(m *Manager, ctx context.Context, ch Channel, ev InboundEvent) error {
		if m.kernel == nil {
			return nil
		}
		if err := m.kernel.ResetSession(); err != nil {
			if errors.Is(err, kernel.ErrResetDuringTurn) {
				_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Cannot reset during active turn — send /stop first.")
			} else {
				_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Session reset failed: "+resetCommandErrorText(err))
			}
			return nil
		}
		key := m.sessionKeyForInbound(ev)
		m.clearSessionBoundaryControlState(key)
		if m.cfg.SessionMap != nil {
			if err := m.cfg.SessionMap.Put(ctx, key, ""); err != nil {
				m.log.Warn("clear session mapping", "key", key, "err", err)
			}
		}
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Session reset. Next message starts fresh.")
		return nil
	},
	EventRestart: func(m *Manager, ctx context.Context, ch Channel, ev InboundEvent) error {
		return m.handleRestartCommand(ctx, ch, ev)
	},
	EventSteer: func(m *Manager, ctx context.Context, ch Channel, ev InboundEvent) error {
		m.handleSteerCommand(ctx, ch, ev)
		return nil
	},
	EventQueue: func(m *Manager, ctx context.Context, ch Channel, ev InboundEvent) error {
		m.handleQueueCommand(ctx, ch, ev)
		return nil
	},
	EventUsage: func(m *Manager, ctx context.Context, ch Channel, ev InboundEvent) error {
		m.handleUsageCommand(ctx, ch, ev)
		return nil
	},
	EventStatus: func(m *Manager, ctx context.Context, ch Channel, ev InboundEvent) error {
		m.handleStatusCommand(ctx, ch, ev)
		return nil
	},
	EventTitle: func(m *Manager, ctx context.Context, ch Channel, ev InboundEvent) error {
		m.handleTitleCommand(ctx, ch, ev)
		return nil
	},
	EventSkills: func(m *Manager, ctx context.Context, ch Channel, ev InboundEvent) error {
		m.handleSkillsCommand(ctx, ch, ev)
		return nil
	},
	EventCommands: func(m *Manager, ctx context.Context, ch Channel, ev InboundEvent) error {
		m.handleCommandsCommand(ctx, ch, ev)
		return nil
	},
	EventVerbose: func(m *Manager, ctx context.Context, ch Channel, ev InboundEvent) error {
		m.handleVerboseCommand(ctx, ch, ev)
		return nil
	},
	EventModel: func(m *Manager, ctx context.Context, ch Channel, ev InboundEvent) error {
		m.handleModelCommand(ctx, ch, ev)
		return nil
	},
	EventSessions: func(m *Manager, ctx context.Context, ch Channel, ev InboundEvent) error {
		m.handleSessionsCommand(ctx, ch, ev)
		return nil
	},
	EventProfile: func(m *Manager, ctx context.Context, ch Channel, ev InboundEvent) error {
		m.handleProfileCommand(ctx, ch, ev)
		return nil
	},
	EventGateway: func(m *Manager, ctx context.Context, ch Channel, ev InboundEvent) error {
		m.handlePlatformsCommand(ctx, ch, ev)
		return nil
	},
	EventPlatformControl: func(m *Manager, ctx context.Context, ch Channel, ev InboundEvent) error {
		m.handlePlatformControlCommand(ctx, ch, ev)
		return nil
	},
	EventReasoning: func(m *Manager, ctx context.Context, ch Channel, ev InboundEvent) error {
		m.handleReasoningCommand(ctx, ch, ev)
		return nil
	},
	EventBusy: func(m *Manager, ctx context.Context, ch Channel, ev InboundEvent) error {
		m.handleBusyCommand(ctx, ch, ev)
		return nil
	},
	EventTTS: func(m *Manager, ctx context.Context, ch Channel, ev InboundEvent) error {
		m.handleTTSCommand(ctx, ch, ev)
		return nil
	},
	EventGoal: func(m *Manager, ctx context.Context, ch Channel, ev InboundEvent) error {
		m.handleGoalCommand(ctx, ch, ev)
		return nil
	},
	EventTopic: func(m *Manager, ctx context.Context, ch Channel, ev InboundEvent) error {
		m.handleTelegramTopicCommand(ctx, ch, ev)
		return nil
	},
	EventKanban: func(m *Manager, ctx context.Context, ch Channel, ev InboundEvent) error {
		m.handleKanbanCommand(ctx, ch, ev)
		return nil
	},
	EventPersonality: func(m *Manager, ctx context.Context, ch Channel, ev InboundEvent) error {
		m.handlePersonalityCommand(ctx, ch, ev)
		return nil
	},
	EventSpawn: func(m *Manager, ctx context.Context, ch Channel, ev InboundEvent) error {
		m.handleSpawnCommand(ctx, ch, ev)
		return nil
	},
	EventReload: func(m *Manager, ctx context.Context, ch Channel, ev InboundEvent) error {
		m.handleReloadCommand(ctx, ch, ev)
		return nil
	},
	EventReloadSkills: func(m *Manager, ctx context.Context, ch Channel, ev InboundEvent) error {
		m.handleReloadSkillsCommand(ctx, ch, ev)
		return nil
	},
	EventRetry: func(m *Manager, ctx context.Context, ch Channel, ev InboundEvent) error {
		m.handleRetryCommand(ctx, ch, ev)
		return nil
	},
	EventUndo: func(m *Manager, ctx context.Context, ch Channel, ev InboundEvent) error {
		m.handleUndoCommand(ctx, ch, ev)
		return nil
	},
	EventCompress: func(m *Manager, ctx context.Context, ch Channel, ev InboundEvent) error {
		m.handleCompressCommand(ctx, ch, ev)
		return nil
	},
}

func (m *Manager) handleCompressCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	compressor, ok := m.kernel.(kernelManualCompressor)
	if !ok || compressor == nil {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "/compress is unavailable: context compression is not wired in this build")
		return
	}
	focus := llm.ParseManualCompressionFocus(ev.Text)
	if err := compressor.ManualCompress(focus); err != nil {
		if errors.Is(err, kernel.ErrCompressDuringTurn) {
			_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Cannot compress during active turn — send /stop first.")
			return
		}
		if errors.Is(err, kernel.ErrCompressionUnavailable) || errors.Is(err, llm.ErrCompressionDisabled) {
			_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "/compress is unavailable: context compression is disabled")
			return
		}
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Session compression failed: "+resetCommandErrorText(err))
		return
	}
	msg := "Session compressed."
	if focus != "" {
		msg += " Focus: " + focus
	}
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, msg)
}

func (m *Manager) handleRetryCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	store := m.cfg.SessionHistoryStore
	if store == nil {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "/retry is unavailable: session history replay is not wired in this build")
		return
	}
	resumer, ok := m.kernel.(kernelSessionResumer)
	if !ok || resumer == nil {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "/retry is unavailable: session history replay is not wired in this build")
		return
	}
	sessionID := m.sessionIDForRetryUndo(ctx, ev)
	if sessionID == "" {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "No previous message to retry.")
		return
	}
	history, err := store.LoadSessionHistory(ctx, sessionID)
	if err != nil || len(history) == 0 {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "No previous message to retry.")
		return
	}
	idx := nthUserMessageIndexFromEnd(history, 1)
	if idx < 0 {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "No previous message to retry.")
		return
	}
	lastUser := strings.TrimSpace(history[idx].Content)
	if lastUser == "" {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "No previous message to retry.")
		return
	}
	truncated := append([]llm.Message(nil), history[:idx]...)
	if err := store.RewriteSessionHistory(ctx, sessionID, truncated); err != nil {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Session retry failed: "+resetCommandErrorText(err))
		return
	}
	if err := resumer.ResumeSession(sessionID, truncated); err != nil {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Session retry failed: "+resetCommandErrorText(err))
		return
	}
	if err := m.kernel.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventSubmit, Text: lastUser, SessionID: sessionID}); err != nil {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Session retry failed: "+resetCommandErrorText(err))
	}
}

func (m *Manager) handleUndoCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	store := m.cfg.SessionHistoryStore
	if store == nil {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "/undo is unavailable: session history replay is not wired in this build")
		return
	}
	resumer, ok := m.kernel.(kernelSessionResumer)
	if !ok || resumer == nil {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "/undo is unavailable: session history replay is not wired in this build")
		return
	}
	sessionID := m.sessionIDForRetryUndo(ctx, ev)
	if sessionID == "" {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Nothing to undo.")
		return
	}
	n := undoTurnCount(ev.Text)
	result, err := store.RewindSessionHistory(ctx, sessionID, n)
	if err != nil {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Nothing to undo.")
		return
	}
	if result.SessionID == "" {
		result.SessionID = sessionID
	}
	if err := resumer.ResumeSession(result.SessionID, result.History); err != nil {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Session undo failed: "+resetCommandErrorText(err))
		return
	}
	turnsUndone := result.TurnsUndone
	if turnsUndone <= 0 {
		turnsUndone = n
	}
	preview := strings.TrimSpace(result.TargetText)
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Removed "+strconv.Itoa(turnsUndone)+" turn(s). Last removed message: "+preview)
}

func (m *Manager) sessionIDForRetryUndo(ctx context.Context, ev InboundEvent) string {
	if m.cfg.SessionMap == nil {
		return ""
	}
	sessionID, err := m.cfg.SessionMap.Get(ctx, m.sessionKeyForInbound(ev))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(sessionID)
}

func undoTurnCount(text string) int {
	fields := strings.Fields(text)
	if len(fields) < 2 {
		return 1
	}
	n, err := strconv.Atoi(fields[1])
	if err != nil || n <= 0 {
		return 1
	}
	return n
}

func nthUserMessageIndexFromEnd(history []llm.Message, n int) int {
	if n <= 0 {
		n = 1
	}
	seen := 0
	for i := len(history) - 1; i >= 0; i-- {
		if strings.TrimSpace(history[i].Role) != "user" {
			continue
		}
		seen++
		if seen == n {
			return i
		}
	}
	return -1
}

func (m *Manager) dispatchGatewayCommandEvent(ctx context.Context, ch Channel, ev InboundEvent) (bool, error) {
	handler, ok := gatewayCommandHandlers[ev.Kind]
	if !ok {
		return false, nil
	}
	return true, handler(m, ctx, ch, ev)
}

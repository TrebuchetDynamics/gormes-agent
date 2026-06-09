package gateway

import (
	"context"
	"errors"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
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
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "/retry is coming soon — session retry is not yet implemented in the gateway")
		return nil
	},
	EventUndo: func(m *Manager, ctx context.Context, ch Channel, ev InboundEvent) error {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "/undo is coming soon — message undo is not yet implemented in the gateway")
		return nil
	},
}

func (m *Manager) dispatchGatewayCommandEvent(ctx context.Context, ch Channel, ev InboundEvent) (bool, error) {
	handler, ok := gatewayCommandHandlers[ev.Kind]
	if !ok {
		return false, nil
	}
	return true, handler(m, ctx, ch, ev)
}

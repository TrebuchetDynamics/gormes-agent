package gateway

import (
	"context"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
)

type noSkillProvider struct{}

func (noSkillProvider) BuildSkillBlock(context.Context, string) (string, []string, error) {
	return "", nil, nil
}

func (m *Manager) handleSubmitEvent(ctx context.Context, ch Channel, ev InboundEvent) {
	if rewritten, consumed := m.preprocessSlashSubmit(ctx, ch, ev); consumed {
		return
	} else {
		ev = rewritten
	}
	if m.kernel == nil && m.cfg.AgentRuntimeFactory == nil {
		return
	}
	if m.dropDuplicateInboundSubmit(ev) {
		return
	}
	queued, full := m.queueFollowUpIfActive(ev)
	if queued {
		return
	}
	if full {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, followUpQueueFullNotice)
		return
	}
	m.pinTurn(ev.Platform, ev.ChatID, ev.MsgID)
	m.submitPinned(ctx, ch, ev)
}

func (m *Manager) preprocessSlashSubmit(ctx context.Context, ch Channel, ev InboundEvent) (InboundEvent, bool) {
	body := strings.TrimSpace(ev.Text)
	if !strings.HasPrefix(body, "/") {
		return ev, false
	}

	cmd, ok := ResolveCommand(body)
	if !ok {
		if rewritten, matched := m.expandSkillSlashSubmit(ctx, ev, body); matched {
			return rewritten, false
		}
		name := slashCommandName(body)
		if isRecognizedUnavailableSlashCommand(name) {
			_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "/"+name+" is recognized but unavailable in this build")
		} else {
			_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, UnknownSlashCommandGuidance(name))
		}
		return ev, true
	}
	if m.hasActiveTurn() && cmd.ActiveTurnPolicy == CommandActiveTurnPolicyReject {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Gormes is busy — finish the current turn or send /stop before /"+cmd.Name)
		return ev, true
	}
	if cmd.ActiveTurnPolicy == CommandActiveTurnPolicyUnavailable {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "/"+cmd.Name+" is recognized but unavailable in this build")
		return ev, true
	}
	commandEvent := ev
	commandEvent.Kind = cmd.Kind
	if slashCommandKindCarriesBody(cmd.Kind) {
		commandEvent.Text = body
	} else {
		commandEvent.Text = ""
	}
	return ev, m.dispatchCommandEvent(ctx, ch, commandEvent)
}

func slashCommandKindCarriesBody(kind EventKind) bool {
	switch kind {
	case EventSteer, EventQueue, EventTitle, EventSessions, EventProfile, EventSkills, EventCommands, EventReasoning,
		EventBusy, EventTTS, EventReload, EventReloadSkills, EventRetry, EventGoal, EventTopic, EventKanban,
		EventSpawn, EventPlatformControl:
		return true
	default:
		return false
	}
}

func (m *Manager) expandSkillSlashSubmit(ctx context.Context, ev InboundEvent, body string) (InboundEvent, bool) {
	if m.cfg.SkillRuntime == nil {
		return ev, false
	}
	commands, _, err := m.cfg.SkillRuntime.SkillSlashCommands(ctx, skills.RuntimeOptions{})
	if err != nil {
		if m.log != nil {
			m.log.Warn("gateway: skill slash command scan failed", "err", err)
		}
		return ev, false
	}
	command, ok := skills.ResolveSkillSlashCommand(commands, body)
	if !ok {
		return ev, false
	}
	_, args := splitGatewayCommandLine(body)
	rewritten := ev
	rewritten.Kind = EventSubmit
	rewritten.Text = skills.BuildSkillSlashCommandMessage(command, args, skills.SlashMessageOptions{RuntimeNote: skillSlashRuntimeNote(ev)})
	rewritten.SkillSlashExpanded = true
	return rewritten, true
}

func skillSlashRuntimeNote(ev InboundEvent) string {
	platform := strings.TrimSpace(ev.Platform)
	if platform == "" {
		return "gateway"
	}
	return platform
}

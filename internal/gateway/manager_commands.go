package gateway

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func (m *Manager) dispatchCommandEvent(ctx context.Context, ch Channel, ev InboundEvent) bool {
	handled, _ := m.dispatchGatewayCommandEvent(ctx, ch, ev)
	if handled {
		return true
	}
	if ev.Kind == EventUnknown {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "unknown command")
		return true
	}
	return false
}

func (m *Manager) handleReasoningCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	reply, err := m.DispatchReasoning(m.sessionKeyForInbound(ev), commandArgs(ev.Text))
	if err != nil {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Reasoning command error: "+reasoningCommandErrorText(err)+"\n\nUsage: /reasoning [low|medium|high|reset|show] [--global]")
		return
	}
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, formatReasoningReply(reply))
}

func reasoningCommandErrorText(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return ""
	}
	replacer := strings.NewReplacer("`", "'", "*", "'", "#", "＃")
	return strings.Join(strings.Fields(replacer.Replace(msg)), " ")
}

func (m *Manager) handleBusyCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	args := strings.Fields(ev.Text)
	mode := m.cfg.BusyInputMode
	if mode == "" {
		mode = "interrupt"
	}

	if len(args) >= 2 {
		switch strings.ToLower(args[1]) {
		case "queue", "q":
			mode = "queue"
		case "steer", "s":
			mode = "steer"
		case "interrupt", "i":
			mode = "interrupt"
		case "", "status", "show":
			_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("⚙ Busy input mode: **%s**\n\n• interrupt — stop current task and respond to new message\n• queue — silently hold message for next turn\n• steer — inject guidance mid-turn", mode))
			return
		default:
			_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "⚠ Usage: /busy [queue|steer|interrupt|status]")
			return
		}
		m.cfg.BusyInputMode = mode
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("✅ Busy input mode set to **%s**", mode))
		return
	}
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("⚙ Busy input mode: **%s**\nUsage: /busy [queue|steer|interrupt|status]", mode))
}

func commandArgs(body string) []string {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil
	}
	fields := strings.Fields(body)
	if len(fields) <= 1 {
		return nil
	}
	return fields[1:]
}

func formatReasoningReply(reply ReasoningReply) string {
	scope := strings.TrimSpace(reply.Scope)
	if scope == "" || scope == ReasoningSourceUnset {
		if reply.PersistFailed {
			return "Reasoning effort: default\n\nGlobal persistence failed; no session override is active."
		}
		return "Reasoning effort: default"
	}
	effort := strings.TrimSpace(string(reply.Effort))
	if effort == "" {
		effort = "default"
	}
	text := "Reasoning effort: " + effort + " (" + scope + ")"
	if reply.PersistFailed {
		text += "\n\nGlobal persistence failed; using a session-only override."
	}
	return text
}

func (m *Manager) handleVerboseCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	if !m.cfg.ToolProgressCommandEnabled {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "The `/verbose` command is not enabled for messaging platforms.\n\nEnable it in Gormes `config.toml`:\n```toml\n[display]\ntool_progress_command = true\n```")
		return
	}
	platform := strings.ToLower(strings.TrimSpace(ev.Platform))
	if platform == "" {
		platform = "unknown"
	}
	mode := nextToolProgressMode(m.toolProgressMode(platform))
	if m.cfg.ToolProgressModes == nil {
		m.cfg.ToolProgressModes = map[string]string{}
	}
	m.cfg.ToolProgressModes[platform] = mode

	text := toolProgressModeDescription(mode)
	if m.cfg.PersistToolProgressMode == nil {
		text += "\n_(could not save to config: persistence unavailable)_"
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, text)
		return
	}
	if err := m.cfg.PersistToolProgressMode(platform, mode); err != nil {
		text += "\n_(could not save to config: " + verboseCommandErrorText(err) + ")_"
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, text)
		return
	}
	text += "\n_(saved for **" + platform + "** — takes effect on next message)_"
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, text)
}

func verboseCommandErrorText(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return ""
	}
	lower := strings.ToLower(msg)
	compact := compactVerboseSecretSeparators(lower)
	for _, marker := range []string{"token", "api_key", "apikey", "authorization", "bearer", "secret", "password"} {
		if strings.Contains(lower, marker) || strings.Contains(compact, marker) {
			return "[redacted]"
		}
	}
	replacer := strings.NewReplacer("`", "'", "*", "'", "#", "＃")
	return strings.Join(strings.Fields(replacer.Replace(msg)), " ")
}

func compactVerboseSecretSeparators(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (m *Manager) handleSessionsCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	if m.cfg.SessionMap == nil {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Sessions are not available in this build.")
		return
	}
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "📋 Use `/status` for details on the current session. Use `/new` to start fresh.")
}

func (m *Manager) handleModelCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	isTelegram := strings.HasPrefix(ch.Name(), "telegram")
	if m.modelPickerResolver != nil && isTelegram {
		resp, err := m.modelPickerResolver.OpenModelPicker(ctx, ModelPickerRequest{ChatID: ev.ChatID})
		if err == nil {
			_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, resp.Text)
			return
		}
	}
	model := "unknown"
	provider := "unknown"
	over := m.modelOverride
	if over.Model != "" {
		model = over.Model
	} else if m.cfg.LiveTurnActiveModel != nil {
		model = m.cfg.LiveTurnActiveModel()
	}
	if over.Provider != "" {
		provider = over.Provider
	} else if m.cfg.LiveTurnActiveProvider != nil {
		provider = m.cfg.LiveTurnActiveProvider()
	}
	if model == "" {
		model = "unknown"
	}
	if provider == "" {
		provider = "unknown"
	}
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("🤖 **Model:** `%s`\n📡 **Provider:** `%s`", modelCommandFieldText(model), modelCommandFieldText(provider)))
}

func modelCommandFieldText(value string) string {
	msg := strings.TrimSpace(value)
	if msg == "" {
		return "unknown"
	}
	replacer := strings.NewReplacer("`", "'", "*", "'", "#", "＃")
	return strings.Join(strings.Fields(replacer.Replace(msg)), " ")
}

func (m *Manager) handleProfileCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	home := config.GormesHome()
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("👤 **Profile:** `(default)`\n📂 **Home:** `%s`", modelCommandFieldText(home)))
}

func (m *Manager) handlePlatformsCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	platforms := m.formatConnectedPlatforms()
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("📡 **Connected Platforms:** %s\nUse `/status` for full session details.", platforms))
}

// handlePlatformControlCommand is the gateway slash-handler port of Hermes
// gateway/run.py:_handle_platform_command (PR #26600): `/platform
// <list|pause|resume> [name]`. The shared platform reconnect/circuit-breaker
// queue is a tested lifecycle seam not yet wired into the live manager (see
// the "Gateway platform reconnect isolation" row's deferred integration), so
// the live failed-platform set is currently empty and pause/resume on a
// non-queued platform truthfully reports it is not in the retry queue.
func (m *Manager) handlePlatformControlCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	m.mu.Lock()
	connected := make(map[string]Channel, len(m.channels))
	for name, channel := range m.channels {
		connected[name] = channel
	}
	m.mu.Unlock()
	// No live failed-platform set is wired into the manager yet; the deferred
	// lifecycle-integration row will pass the real queue here.
	reply := HandlePlatformCommand(ev.Text, connected, nil)
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, reply)
}

func (m *Manager) formatConnectedPlatforms() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	names := make([]string, 0, len(m.channels))
	for name := range m.channels {
		names = append(names, modelCommandFieldText(name))
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "none"
	}
	return strings.Join(names, ", ")
}

func nextToolProgressMode(current string) string {
	switch normalizeGatewayToolProgressMode(current) {
	case "off":
		return "new"
	case "new":
		return "all"
	case "all":
		return "verbose"
	default:
		return "off"
	}
}

func toolProgressModeDescription(mode string) string {
	switch normalizeGatewayToolProgressMode(mode) {
	case "off":
		return "⚙️ Tool progress: **OFF** — no tool activity shown."
	case "new":
		return "⚙️ Tool progress: **NEW** — shown when tool changes (preview length: `display.tool_preview_length`, default 40)."
	case "verbose":
		return "⚙️ Tool progress: **VERBOSE** — every tool call with safe bounded arguments."
	default:
		return "⚙️ Tool progress: **ALL** — every tool call shown (preview length: `display.tool_preview_length`, default 40)."
	}
}

func (m *Manager) handleSteerCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	parsed := ParseSteerCommand(ev.Text, steerPayloadMetadataFromInbound(ev))
	if parsed.Evidence != "" {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, string(parsed.Evidence))
		return
	}

	if m.hasActiveTurn() {
		if m.kernel != nil {
			if err := m.kernel.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventSteer, Text: parsed.Guidance}); err == nil {
				_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, string(SteerEvidenceInjected)+": pending for next tool-result boundary; "+string(SteerEvidencePreview)+": "+parsed.Preview)
				return
			}
		}
	}

	followUp := ev
	followUp.Kind = EventSubmit
	followUp.Text = parsed.Guidance
	followUp.Attachments = nil
	queued, full := m.queueFollowUpIfActive(followUp)
	if full {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, followUpQueueFullNotice)
		return
	}
	if queued {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, string(SteerEvidenceUnavailable)+": mid-run injection unavailable; "+string(SteerEvidenceQueued)+"; "+string(SteerEvidencePreview)+": "+parsed.Preview)
		return
	}
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, string(SteerEvidenceUnavailable)+": no active turn; "+string(SteerEvidencePreview)+": "+parsed.Preview)
}

func (m *Manager) handleQueueCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	text := strings.TrimSpace(strings.Join(commandArgs(ev.Text), " "))
	if text == "" {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Usage: /queue <prompt>")
		return
	}
	followUp := ev
	followUp.Kind = EventSubmit
	followUp.Text = text
	followUp.Attachments = nil
	queued, full := m.queueFollowUpIfActive(followUp)
	if full {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, followUpQueueFullNotice)
		return
	}
	if !queued {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "queue_unavailable: no active turn; send the prompt without /queue to run it now")
		return
	}
	depth := m.followUpQueueDepth()
	if depth <= 1 {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Queued for the next turn.")
		return
	}
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, fmt.Sprintf("Queued for the next turn. (%d queued)", depth))
}

func steerPayloadMetadataFromInbound(ev InboundEvent) SteerPayloadMetadata {
	meta := SteerPayloadMetadata{AttachmentCount: len(ev.Attachments)}
	for _, attachment := range ev.Attachments {
		kind := strings.ToLower(strings.TrimSpace(attachment.Kind + " " + attachment.MediaType))
		if strings.Contains(kind, "image") {
			meta.ImageCount++
		}
	}
	return meta
}

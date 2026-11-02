package gateway

import (
	"context"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/voicemode"
)

const voiceUsageLine = "Usage: /voice [on|off|tts|status]"
const voiceSubcommandHint = "Subcommands: /voice on, /voice off, /voice tts, /voice status"

const (
	voiceReplyEnabledVoiceOnly = "Voice mode enabled (voice only). Replies to voice messages will be spoken."
	voiceReplyEnabledAll       = "Voice mode enabled. Every reply will be spoken."
	voiceReplyDisabled         = "Voice mode disabled. Replies will be text only."
	voiceReplyPersistFailed    = "Voice mode update failed; the previous mode is still active."
	voiceReplyChannelsMissing  = "Live voice channels are not available in this build. Use /voice on, /voice off, /voice tts, or /voice status."
)

func (m *Manager) handleVoiceCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	if m == nil || m.voiceModeStore == nil {
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, voiceReplyChannelsMissing)
		return
	}
	chatKey := ev.ChatKey()
	current, found, err := m.voiceModeStore.Get(ctx, chatKey)
	if err != nil {
		m.log.Warn("read voice mode", "chat_key", chatKey, "err", err)
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, voiceReplyPersistFailed)
		return
	}
	if !found {
		current = voicemode.ModeOff
	}

	args := strings.Fields(ev.Text)
	sub := ""
	if len(args) > 1 {
		sub = strings.ToLower(args[1])
	}

	switch sub {
	case "on", "enable":
		m.setVoiceModeReply(ctx, ch, ev.ChatID, chatKey, voicemode.ModeVoiceOnly, voiceReplyEnabledVoiceOnly)
	case "off", "disable":
		m.setVoiceModeReply(ctx, ch, ev.ChatID, chatKey, voicemode.ModeOff, voiceReplyDisabled)
	case "tts":
		m.setVoiceModeReply(ctx, ch, ev.ChatID, chatKey, voicemode.ModeAll, voiceReplyEnabledAll)
	case "status":
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, "Voice mode: "+current.Label())
	case "":
		// Bare /voice toggles: off -> voice_only, everything else -> off.
		if current == voicemode.ModeOff {
			m.setVoiceModeReply(ctx, ch, ev.ChatID, chatKey, voicemode.ModeVoiceOnly, voiceReplyEnabledVoiceOnly+"\n\n"+voiceSubcommandHint)
			return
		}
		m.setVoiceModeReply(ctx, ch, ev.ChatID, chatKey, voicemode.ModeOff, voiceReplyDisabled+"\n\n"+voiceSubcommandHint)
	case "channel", "join", "leave":
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, voiceReplyChannelsMissing)
	default:
		_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, voiceUsageLine)
	}
}

func (m *Manager) setVoiceModeReply(ctx context.Context, ch Channel, chatID, chatKey string, mode voicemode.Mode, reply string) {
	if err := m.voiceModeStore.Set(ctx, chatKey, mode); err != nil {
		m.log.Warn("persist voice mode", "chat_key", chatKey, "err", err)
		_, _ = m.sendWithHooks(ctx, ch, chatID, voiceReplyPersistFailed)
		return
	}
	_, _ = m.sendWithHooks(ctx, ch, chatID, reply)
}

// turnVoiceMode resolves the voice mode for the active turn. It prefers the
// chat key captured when the turn was pinned and falls back to deliveryKey so
// direct delivery callers (and tests) can resolve a chat-scoped mode.
func (m *Manager) turnVoiceMode(ctx context.Context, deliveryKey string) (voicemode.Mode, bool) {
	if m == nil || m.voiceModeStore == nil {
		return "", false
	}
	m.turnMu.Lock()
	chatKey := m.turnVoiceChatKey
	m.turnMu.Unlock()
	if strings.TrimSpace(chatKey) == "" {
		chatKey = deliveryKey
	}
	if strings.TrimSpace(chatKey) == "" {
		return "", false
	}
	mode, found, err := m.voiceModeStore.Get(ctx, chatKey)
	if err != nil {
		m.log.Warn("read voice mode for delivery", "chat_key", chatKey, "err", err)
		return "", false
	}
	return mode, found
}

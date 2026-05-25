package gateway

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

const audioDeliveryGuidance = "## Audio Delivery\nGateway audio delivery is enabled for this turn. Answer normally in written text; the gateway will synthesize and attach the audio after your final answer. Do not claim that you generated audio yourself, and do not claim the TTS provider is unavailable."

var audioIntentTextReplacer = strings.NewReplacer(
	"á", "a",
	"é", "e",
	"í", "i",
	"ó", "o",
	"ú", "u",
	"ü", "u",
)

func inboundRequestsAudioReply(ev InboundEvent) bool {
	for _, attachment := range ev.Attachments {
		switch strings.ToLower(strings.TrimSpace(attachment.Kind)) {
		case "voice", "audio", "voice_transcript":
			return true
		}
	}
	text := audioIntentTextReplacer.Replace(strings.ToLower(strings.Join(strings.Fields(ev.Text), " ")))
	if text == "" {
		return false
	}
	for _, phrase := range []string{
		"cannot read",
		"can't read",
		"cant read",
		"audio version",
		"send audio",
		"voice reply",
		"reply by voice",
		"read it aloud",
		"read out loud",
		"mandame audio",
		"mandamelo en audio",
		"mandalo en audio",
		"enviame audio",
		"enviamelo en audio",
		"envialo en audio",
		"pasame audio",
		"pasamelo en audio",
		"por audio",
		"para audio",
		"audio por favor",
		"leelo en voz alta",
		"leemelo",
		"voz alta",
	} {
		if strings.Contains(text, phrase) {
			return true
		}
	}
	return false
}

func appendAudioDeliveryGuidance(sessionBlock string, enabled bool) string {
	if !enabled {
		return sessionBlock
	}
	if strings.TrimSpace(sessionBlock) == "" {
		return audioDeliveryGuidance
	}
	return strings.TrimRight(sessionBlock, "\n") + "\n\n" + audioDeliveryGuidance
}

func (m *Manager) formatFinalDeliveryForTurn(ctx context.Context, platform string, f kernel.RenderFrame, sessionKey string, audioRequested bool) (string, []OutboundMedia) {
	content := PrepareMediaDeliveryContent(FinalAssistantText(f))
	text := content.Text
	media := m.appendAutoTTSMedia(ctx, sessionKey, text, content.Media, audioRequested)
	if strings.TrimSpace(text) == "" && len(media) > 0 {
		text = "Media attached."
	}
	if isTelegramPlatform(platform) {
		return FormatFinalTelegramText(text), media
	}
	return FormatFinalPlainText(text), media
}

func (m *Manager) formatFinalDeliveryPagesForTurn(ctx context.Context, platform string, f kernel.RenderFrame, sessionKey string, audioRequested bool) ([]string, []OutboundMedia) {
	text, media := m.formatFinalDeliveryForTurn(ctx, platform, f, sessionKey, audioRequested)
	if isTelegramPlatform(platform) {
		return paginateTelegramText(text), media
	}
	return paginatePlainText(text), media
}

func (m *Manager) appendAutoTTSMedia(ctx context.Context, sessionKey, text string, media []OutboundMedia, audioRequested bool) []OutboundMedia {
	if strings.TrimSpace(text) == "" || hasAudioMedia(media) {
		return media
	}
	if m == nil || m.cfg.ToolRegistry == nil {
		return media
	}
	cfg := m.getTTSConfig(sessionKey)
	if !audioRequested && !cfg.Enabled {
		return media
	}
	tool, ok := m.cfg.ToolRegistry.Get("text_to_speech")
	if !ok || tool == nil {
		return media
	}
	args, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return media
	}
	raw, err := tool.Execute(ctx, args)
	if err != nil {
		m.log.Warn("gateway auto TTS failed", "err", err)
		return media
	}
	var result struct {
		Success  bool   `json:"success"`
		FilePath string `json:"file_path"`
		MediaTag string `json:"media_tag"`
		Evidence string `json:"evidence"`
		Error    string `json:"error"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		m.log.Warn("gateway auto TTS result invalid", "err", err)
		return media
	}
	if !result.Success {
		m.log.Warn("gateway auto TTS unavailable", "evidence", result.Evidence, "err", result.Error)
		return media
	}
	content := PrepareMediaDeliveryContent(result.MediaTag)
	if len(content.Media) == 0 && strings.TrimSpace(result.FilePath) != "" {
		content = PrepareMediaDeliveryContent("MEDIA:" + result.FilePath)
	}
	if len(content.Media) == 0 {
		m.log.Warn("gateway auto TTS produced no deliverable media", "evidence", result.Evidence)
		return media
	}
	return append(media, content.Media...)
}

func hasAudioMedia(media []OutboundMedia) bool {
	for _, item := range media {
		if ClassifyOutboundMedia(item) == OutboundMediaKindAudio {
			return true
		}
	}
	return false
}

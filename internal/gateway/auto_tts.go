package gateway

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/audiodelivery"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

const audioDeliveryGuidance = audiodelivery.Guidance

func inboundRequestsAudioReply(ev InboundEvent) bool {
	kinds := make([]string, 0, len(ev.Attachments))
	for _, attachment := range ev.Attachments {
		kinds = append(kinds, attachment.Kind)
	}
	return audiodelivery.RequestsAudioReply(ev.Text, kinds)
}

func appendAudioDeliveryGuidance(sessionBlock string, enabled bool) string {
	return audiodelivery.AppendGuidance(sessionBlock, enabled)
}

func (m *Manager) formatFinalDeliveryForTurn(ctx context.Context, platform string, f kernel.RenderFrame, sessionKey string, audioRequested bool) (string, []OutboundMedia) {
	content := PrepareMediaDeliveryContent(FinalAssistantText(f))
	text := content.Text
	media := m.appendAutoTTSMedia(ctx, platform, sessionKey, text, content.Media, audioRequested)
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

func (m *Manager) appendAutoTTSMedia(ctx context.Context, platform, sessionKey, text string, media []OutboundMedia, audioRequested bool) []OutboundMedia {
	if strings.TrimSpace(text) == "" || hasAudioMedia(media) {
		return media
	}
	if m == nil || m.cfg.ToolRegistry == nil {
		return media
	}
	cfg := m.getTTSConfig(sessionKey)
	if cfg.Engine == TTSEngineDisabled {
		return media
	}
	if !audioRequested && !cfg.Enabled {
		return media
	}
	tool, ok := m.cfg.ToolRegistry.Get("text_to_speech")
	if !ok || tool == nil {
		return media
	}
	toolArgs := map[string]string{
		"text":     text,
		"platform": platform,
	}
	if cfg.Engine != "" {
		toolArgs["provider"] = string(cfg.Engine)
	}
	if strings.TrimSpace(cfg.Voice) != "" {
		toolArgs["voice"] = cfg.Voice
	}
	if cfg.Speed != "" {
		toolArgs["speed"] = string(cfg.Speed)
	}
	if strings.TrimSpace(cfg.Language) != "" {
		toolArgs["language"] = cfg.Language
	}
	args, err := json.Marshal(toolArgs)
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

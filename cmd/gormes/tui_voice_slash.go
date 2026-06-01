package main

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func newTUIVoiceToggleFunc(cfg config.Config) tui.VoiceToggleFunc {
	state := &tuiVoiceToggleState{
		recordKey: strings.TrimSpace(cfg.Voice.RecordKey),
		details:   tuiVoiceRequirementsDetails(cfg),
	}
	if state.recordKey == "" {
		state.recordKey = "ctrl+b"
	}
	return state.toggle
}

type tuiVoiceToggleState struct {
	enabled   bool
	tts       bool
	recordKey string
	details   string
}

func (s *tuiVoiceToggleState) toggle(req tui.VoiceToggleRequest) (tui.VoiceToggleResult, error) {
	switch strings.ToLower(strings.TrimSpace(req.Action)) {
	case "", "status", "record":
		// Read-only in the native TUI: live microphone capture is not wired here yet.
	case "on":
		s.enabled = true
	case "off":
		s.enabled = false
		s.tts = false
	case "tts":
		s.tts = !s.tts
	default:
		return tui.VoiceToggleResult{}, fmt.Errorf("unsupported action %q", req.Action)
	}
	return tui.VoiceToggleResult{
		Enabled:   s.enabled,
		TTS:       s.tts,
		RecordKey: s.recordKey,
		Details:   s.details,
	}, nil
}

func tuiVoiceRequirementsDetails(cfg config.Config) string {
	tts := "TTS: not configured"
	if provider := strings.TrimSpace(configuredTTSProvider(cfg)); provider != "" {
		tts = "TTS: configured (" + provider + ")"
	}
	return strings.Join([]string{
		"Audio: unavailable in native TUI (live microphone capture is not started by /voice slash)",
		configuredSTTStatusLine(cfg),
		tts,
	}, "\n")
}

func configuredSTTStatusLine(cfg config.Config) string {
	provider := strings.ToLower(strings.TrimSpace(cfg.STT.Provider))
	if provider == "" {
		return "STT: not configured"
	}
	parts := []string{"STT: configured (" + provider + ")"}
	if model := strings.TrimSpace(cfg.STT.Local.Model); model != "" && (provider == "local" || provider == "local_command") {
		parts = append(parts, "model "+model)
	}
	if language := strings.TrimSpace(cfg.STT.Local.Language); language != "" {
		parts = append(parts, "language "+language)
	}
	return strings.Join(parts, ", ")
}

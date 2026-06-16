package tuiapp

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func TestTUIVoiceSlashBindingLocalModelReceivesVoiceToggle(t *testing.T) {
	setupNativeTUITestEnv(t)
	if err := config.WriteTOMLValue(config.ConfigPath(), "voice.record_key", "ctrl+space"); err != nil {
		t.Fatalf("write voice.record_key: %v", err)
	}
	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}
	var sawToggle bool
	var status, enabled, ttsOn, ttsOff, disabled tui.VoiceToggleResult
	runOfflineTUIForTest(t, cfg, func(model tea.Model) {
		toggle := capturedTUIVoiceToggle(t, model)
		if toggle == nil {
			return
		}
		sawToggle = true
		status, err = toggle(tui.VoiceToggleRequest{Action: "status", SessionID: "sess-voice"})
		if err != nil {
			return
		}
		enabled, err = toggle(tui.VoiceToggleRequest{Action: "on", SessionID: "sess-voice"})
		if err != nil {
			return
		}
		ttsOn, err = toggle(tui.VoiceToggleRequest{Action: "tts", SessionID: "sess-voice"})
		if err != nil {
			return
		}
		ttsOff, err = toggle(tui.VoiceToggleRequest{Action: "tts", SessionID: "sess-voice"})
		if err != nil {
			return
		}
		disabled, err = toggle(tui.VoiceToggleRequest{Action: "off", SessionID: "sess-voice"})
	})
	if !sawToggle {
		t.Fatal("local TUI VoiceToggle = nil, want config-backed /voice adapter")
	}
	if err != nil {
		t.Fatalf("VoiceToggle: %v", err)
	}
	if status.Enabled || status.TTS || status.RecordKey != "ctrl+space" {
		t.Fatalf("status result = %+v, want disabled, TTS off, ctrl+space", status)
	}
	for _, want := range []string{"Audio:", "STT:", "TTS:"} {
		if !strings.Contains(status.Details, want) {
			t.Fatalf("status details missing %q: %q", want, status.Details)
		}
	}
	if !enabled.Enabled || enabled.TTS || enabled.RecordKey != "ctrl+space" {
		t.Fatalf("on result = %+v, want enabled voice without TTS and ctrl+space", enabled)
	}
	if !ttsOn.Enabled || !ttsOn.TTS {
		t.Fatalf("ttsOn result = %+v, want enabled with TTS", ttsOn)
	}
	if !ttsOff.Enabled || ttsOff.TTS {
		t.Fatalf("ttsOff result = %+v, want enabled with TTS disabled", ttsOff)
	}
	if disabled.Enabled || disabled.TTS {
		t.Fatalf("off result = %+v, want disabled voice and TTS", disabled)
	}
}

func TestTUIVoiceSlashBindingRemoteTUIUnchanged(t *testing.T) {
	model := newPlainRemoteTUIModel()
	if toggle := capturedTUIVoiceToggle(t, model); toggle != nil {
		t.Fatal("plain/remote TUI VoiceToggle is non-nil; only local startup should inject /voice adapter")
	}
}

func capturedTUIVoiceToggle(t *testing.T, model tea.Model) tui.VoiceToggleFunc {
	t.Helper()
	return capturedOptionalTUIModelField[tui.VoiceToggleFunc](t, model, "voiceToggle")
}

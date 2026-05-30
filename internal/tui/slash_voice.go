package tui

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/voice"
)

func voiceSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "voice: TUI unavailable"}
	}
	action := parseVoiceSlashAction(input)
	result := VoiceToggleResult{Enabled: false, TTS: false, RecordKey: tools.DefaultVoiceRecordKey, Details: "voice adapter unavailable"}
	updateRecordKey := false
	if model.voiceToggle != nil {
		got, err := model.voiceToggle(VoiceToggleRequest{Action: action, SessionID: model.SessionID()})
		if err != nil {
			model.transientPage = nil
			return SlashResult{Handled: true, StatusMessage: "voice: " + err.Error()}
		}
		result = got
		updateRecordKey = strings.TrimSpace(result.RecordKey) != ""
	}
	if updateRecordKey {
		binding := tools.ResolveVoiceRecordKey(result.RecordKey, tools.VoiceRecordKeyOptions{})
		model.voiceRecordKey = binding.Raw
	}
	lines := renderVoiceToggleLines(action, result)
	if len(lines) == 0 {
		lines = []string{"voice: no status"}
	}
	body := strings.Join(lines, "\n")
	model.transientPage = &TransientPageState{Title: "Voice", Body: body}
	return SlashResult{Handled: true, StatusMessage: firstNonEmptyString(lines...)}
}

func parseVoiceSlashAction(input string) string {
	return voice.Action(input)
}

func renderVoiceToggleLines(action string, result VoiceToggleResult) []string {
	return voice.Lines(action, result)
}

func onOff(value bool) string {
	return voice.OnOff(value)
}

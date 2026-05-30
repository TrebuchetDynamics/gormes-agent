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
	var toggle voice.ToggleFunc
	if model.voiceToggle != nil {
		toggle = voice.ToggleFunc(model.voiceToggle)
	}
	result := voice.HandleSlash(input, model.SessionID(), toggle)
	if result.Err != nil {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: result.StatusMessage}
	}
	if result.UpdateRecordKey {
		binding := tools.ResolveVoiceRecordKey(result.RecordKey, tools.VoiceRecordKeyOptions{})
		model.voiceRecordKey = binding.Raw
	}
	body := strings.Join(result.Lines, "\n")
	model.transientPage = &TransientPageState{Title: "Voice", Body: body}
	return SlashResult{Handled: true, StatusMessage: result.StatusMessage}
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

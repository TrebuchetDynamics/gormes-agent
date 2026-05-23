package tui

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
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
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) < 2 {
		return "status"
	}
	switch strings.ToLower(strings.TrimSpace(fields[1])) {
	case "on", "off", "tts", "status":
		return strings.ToLower(strings.TrimSpace(fields[1]))
	default:
		return "status"
	}
}

func renderVoiceToggleLines(action string, result VoiceToggleResult) []string {
	recordKey := strings.TrimSpace(result.RecordKey)
	if recordKey == "" {
		recordKey = tools.DefaultVoiceRecordKey
	}
	recordKeyLabel := tools.FormatVoiceRecordKeyForStatus(recordKey)
	switch action {
	case "tts":
		if result.TTS {
			return []string{"Voice TTS enabled."}
		}
		return []string{"Voice TTS disabled."}
	case "on", "off":
		if result.Enabled {
			ttsSuffix := ""
			if result.TTS {
				ttsSuffix = " (TTS enabled)"
			}
			return []string{
				"Voice mode enabled" + ttsSuffix,
				fmt.Sprintf("  %s to start/stop recording", recordKeyLabel),
				"  /voice tts  to toggle speech output",
				"  /voice off  to disable voice mode",
			}
		}
		return []string{"Voice mode disabled."}
	default:
		lines := []string{
			"Voice Mode Status",
			fmt.Sprintf("  Mode:       %s", onOff(result.Enabled)),
			fmt.Sprintf("  TTS:        %s", onOff(result.TTS)),
			fmt.Sprintf("  Record key: %s", recordKeyLabel),
		}
		if strings.TrimSpace(result.Details) != "" {
			lines = append(lines, "", "  Requirements:")
			for _, line := range strings.Split(result.Details, "\n") {
				if trimmed := strings.TrimSpace(line); trimmed != "" {
					lines = append(lines, "    "+trimmed)
				}
			}
		}
		return lines
	}
}

func onOff(value bool) string {
	if value {
		return "ON"
	}
	return "OFF"
}

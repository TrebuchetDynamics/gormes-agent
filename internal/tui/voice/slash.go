package voice

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

type Result struct {
	Enabled   bool
	TTS       bool
	RecordKey string
	Details   string
}

func Action(input string) string {
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

func Lines(action string, result Result) []string {
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
			fmt.Sprintf("  Mode:       %s", OnOff(result.Enabled)),
			fmt.Sprintf("  TTS:        %s", OnOff(result.TTS)),
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

func OnOff(value bool) string {
	if value {
		return "ON"
	}
	return "OFF"
}

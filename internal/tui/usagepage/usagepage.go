package usagepage

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/transientpage"
)

const AccountLoadingLine = "Provider account usage: loading..."

// Build renders the read-only TUI usage page from local frame telemetry.
func Build(frame kernel.RenderFrame, sessionID string) (transientpage.State, bool) {
	if !TelemetryPresent(frame.Telemetry) {
		return transientpage.State{}, false
	}
	if strings.TrimSpace(sessionID) == "" {
		sessionID = frame.SessionID
	}
	model := strings.TrimSpace(frame.Model)
	if model == "" {
		model = strings.TrimSpace(frame.Telemetry.Model)
	}
	if model == "" {
		model = "unknown"
	}
	if strings.TrimSpace(sessionID) == "" {
		sessionID = "unknown"
	}

	in := nonNegativeInt(frame.Telemetry.TokensInTotal)
	out := nonNegativeInt(frame.Telemetry.TokensOutTotal)
	lines := []string{
		"Usage source: local TUI frame",
		"Model: " + model,
		"Session: " + sessionID,
		fmt.Sprintf("Input tokens: %d", in),
		fmt.Sprintf("Output tokens: %d", out),
		fmt.Sprintf("Total tokens: %d", in+out),
	}
	if frame.Telemetry.LatencyMsLast > 0 {
		lines = append(lines, fmt.Sprintf("Last latency: %d ms", frame.Telemetry.LatencyMsLast))
	}
	if frame.Telemetry.TokensPerSec > 0 {
		lines = append(lines, fmt.Sprintf("Speed: %.2f tokens/sec", frame.Telemetry.TokensPerSec))
	}
	if frame.ContextStatus != nil {
		if line := ContextLine(*frame.ContextStatus); line != "" {
			lines = append(lines, line)
		}
		if frame.ContextStatus.CompressionCount > 0 {
			lines = append(lines, fmt.Sprintf("Compressions: %d", frame.ContextStatus.CompressionCount))
		}
	}
	return transientpage.State{Title: "Usage", Body: strings.Join(lines, "\n")}, true
}

func AppendAccountLines(body string, lines []string) string {
	section := strings.Join(lines, "\n")
	if strings.TrimSpace(body) == "" {
		return section
	}
	return body + "\n\n" + section
}

func ReplaceAccountLoading(body string, lines []string) string {
	section := strings.Join(lines, "\n")
	if strings.Contains(body, AccountLoadingLine) {
		return strings.Replace(body, AccountLoadingLine, section, 1)
	}
	return AppendAccountLines(body, lines)
}

func TelemetryPresent(t telemetry.Snapshot) bool {
	return t.TokensInTotal > 0 || t.TokensOutTotal > 0
}

func ContextLine(status llm.ContextStatus) string {
	if status.ContextLength <= 0 && status.LastTotalTokens <= 0 {
		return ""
	}
	line := fmt.Sprintf("Context: %d / %d tokens", nonNegativeInt(status.LastTotalTokens), nonNegativeInt(status.ContextLength))
	if status.UsagePercent > 0 {
		line += fmt.Sprintf(" (%.1f%%)", status.UsagePercent)
	}
	return line
}

func nonNegativeInt(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

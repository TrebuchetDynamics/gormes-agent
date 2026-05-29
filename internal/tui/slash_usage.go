package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
)

const (
	usageAccountTimeout     = 30 * time.Second
	usageAccountLoadingLine = "Provider account usage: loading..."
)

type usageAccountMsg struct {
	Lines []string
	Err   error
}

func usageSlashHandler(_ string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "usage: TUI unavailable"}
	}
	page, ok := BuildUsagePage(model.frame, model.SessionID())
	if !ok {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: "no API calls yet"}
	}
	if model.accountUsage != nil {
		page.Body = appendUsageAccountLines(page.Body, []string{usageAccountLoadingLine})
	}
	model.transientPage = &page
	return SlashResult{Handled: true, StatusMessage: "usage opened", Cmd: model.usageAccountCmd()}
}

func (m *Model) usageAccountCmd() tea.Cmd {
	fn := m.accountUsage
	if fn == nil {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), usageAccountTimeout)
		defer cancel()
		snapshot, err := fn(ctx)
		if err != nil {
			return usageAccountMsg{Err: err}
		}
		return usageAccountMsg{Lines: hermes.RenderAccountUsageLines(snapshot, hermes.AccountUsageRenderOptions{})}
	}
}

func (m *Model) handleUsageAccount(msg usageAccountMsg) {
	lines := msg.Lines
	if msg.Err != nil {
		lines = []string{"Provider: unavailable", "Usage unavailable: " + msg.Err.Error()}
		m.statusMessage = "usage account unavailable: " + msg.Err.Error()
	} else {
		m.statusMessage = "usage account updated"
	}
	if m.transientPage == nil || m.transientPage.Title != "Usage" {
		return
	}
	m.transientPage.Body = replaceUsageAccountLoading(m.transientPage.Body, lines)
}

func BuildUsagePage(frame kernel.RenderFrame, sessionID string) (TransientPageState, bool) {
	if !usageTelemetryPresent(frame.Telemetry) {
		return TransientPageState{}, false
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
		if line := usageContextLine(*frame.ContextStatus); line != "" {
			lines = append(lines, line)
		}
		if frame.ContextStatus.CompressionCount > 0 {
			lines = append(lines, fmt.Sprintf("Compressions: %d", frame.ContextStatus.CompressionCount))
		}
	}
	return TransientPageState{Title: "Usage", Body: strings.Join(lines, "\n")}, true
}

func appendUsageAccountLines(body string, lines []string) string {
	section := strings.Join(lines, "\n")
	if strings.TrimSpace(body) == "" {
		return section
	}
	return body + "\n\n" + section
}

func replaceUsageAccountLoading(body string, lines []string) string {
	section := strings.Join(lines, "\n")
	if strings.Contains(body, usageAccountLoadingLine) {
		return strings.Replace(body, usageAccountLoadingLine, section, 1)
	}
	return appendUsageAccountLines(body, lines)
}

func usageTelemetryPresent(t telemetry.Snapshot) bool {
	return t.TokensInTotal > 0 || t.TokensOutTotal > 0
}

func usageContextLine(status hermes.ContextStatus) string {
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

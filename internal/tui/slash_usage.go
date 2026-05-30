package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/usagepage"
)

const (
	usageAccountTimeout     = 30 * time.Second
	usageAccountLoadingLine = usagepage.AccountLoadingLine
)

type usageAccountMsg struct {
	Lines []string
	Err   error
}

func usageSlashHandler(_ string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "usage: TUI unavailable"}
	}
	result := usagepage.HandleSlash(model.frame, model.SessionID(), model.accountUsage != nil)
	if !result.OpenPage {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: result.Status}
	}
	model.transientPage = &result.Page
	var cmd tea.Cmd
	if result.FetchAccount {
		cmd = model.usageAccountCmd()
	}
	return SlashResult{Handled: true, StatusMessage: result.Status, Cmd: cmd}
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
		return usageAccountMsg{Lines: llm.RenderAccountUsageLines(snapshot, llm.AccountUsageRenderOptions{})}
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
	return usagepage.Build(frame, sessionID)
}

func appendUsageAccountLines(body string, lines []string) string {
	return usagepage.AppendAccountLines(body, lines)
}

func replaceUsageAccountLoading(body string, lines []string) string {
	return usagepage.ReplaceAccountLoading(body, lines)
}

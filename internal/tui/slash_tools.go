package tui

import (
	"fmt"
	"strings"
)

func toolsSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "tools: TUI unavailable"}
	}
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) < 2 {
		return slashFallbackResult(input)
	}
	action := strings.ToLower(fields[1])
	if action != "enable" && action != "disable" {
		return slashFallbackResult(input)
	}
	names := append([]string(nil), fields[2:]...)
	if len(names) == 0 {
		return SlashResult{Handled: true, StatusMessage: toolsSlashUsage(action)}
	}
	if model.toolsConfigure == nil {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: "tools: configuration unavailable"}
	}
	result, err := model.toolsConfigure(ToolsConfigureRequest{
		Action:    action,
		Names:     names,
		SessionID: model.SessionID(),
	})
	if err != nil {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: "tools: " + err.Error()}
	}
	lines := renderToolsConfigureLines(action, result)
	if len(lines) == 0 {
		lines = []string{"tools: no changes"}
	}
	body := strings.Join(lines, "\n")
	model.transientPage = &TransientPageState{Title: "Tools", Body: body}
	return SlashResult{Handled: true, StatusMessage: firstNonEmptyString(lines...)}
}

func toolsSlashUsage(action string) string {
	return strings.Join([]string{
		fmt.Sprintf("usage: /tools %s <name> [name ...]", action),
		fmt.Sprintf("built-in toolset: /tools %s web", action),
		fmt.Sprintf("MCP tool: /tools %s github:create_issue", action),
	}, "\n")
}

func renderToolsConfigureLines(action string, result ToolsConfigureResult) []string {
	lines := make([]string, 0, 4)
	if len(result.Changed) > 0 {
		verb := "enabled"
		if action == "disable" {
			verb = "disabled"
		}
		lines = append(lines, fmt.Sprintf("%s: %s", verb, strings.Join(result.Changed, ", ")))
	}
	if len(result.Unknown) > 0 {
		lines = append(lines, "unknown toolsets: "+strings.Join(result.Unknown, ", "))
	}
	if len(result.MissingServers) > 0 {
		lines = append(lines, "missing MCP servers: "+strings.Join(result.MissingServers, ", "))
	}
	if result.Reset {
		lines = append(lines, "session reset. new tool configuration is active.")
	}
	return lines
}

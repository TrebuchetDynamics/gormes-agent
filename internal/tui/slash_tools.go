package tui

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/toolsview"

func toolsSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "tools: TUI unavailable"}
	}
	var configure toolsview.ConfigureFunc
	if model.toolsConfigure != nil {
		configure = func(action string, names []string) (toolsview.Result, error) {
			return model.toolsConfigure(ToolsConfigureRequest{
				Action:    action,
				Names:     names,
				SessionID: model.SessionID(),
			})
		}
	}
	res := toolsview.HandleSlash(input, configure)
	if res.Fallback {
		return slashFallbackResult(input)
	}
	if !res.Open {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: res.Status}
	}
	model.transientPage = &TransientPageState{Title: res.Title, Body: res.Body}
	return SlashResult{Handled: true, StatusMessage: res.Status}
}

func toolsSlashUsage(action string) string {
	return toolsview.Usage(action)
}

func renderToolsConfigureLines(action string, result ToolsConfigureResult) []string {
	return toolsview.Lines(action, result)
}

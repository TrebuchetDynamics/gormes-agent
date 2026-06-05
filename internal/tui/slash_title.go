package tui

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/title"

func titleSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "title: TUI unavailable"}
	}
	res := title.HandleSlash(title.SlashRequest{
		Input:     input,
		SessionID: model.SessionID(),
		TitleFunc: titleSessionFunc(model.sessionTitle),
	})
	return SlashResult{Handled: true, StatusMessage: res.StatusMessage}
}

func titleSessionFunc(fn SessionTitleFunc) title.SessionTitleFunc {
	if fn == nil {
		return nil
	}
	return func(sessionID, nextTitle string) (title.SessionTitleResult, error) {
		res, err := fn(sessionID, nextTitle)
		return title.SessionTitleResult{Title: res.Title, Pending: res.Pending}, err
	}
}

func titleSlashArg(input string) (string, bool) {
	return title.SlashArg(input)
}

package tui

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/reset"

func sessionResetSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: sessionResetSlashKind(input) + ": TUI unavailable"}
	}
	var resetFn reset.Func
	if model.sessionReset != nil {
		resetFn = func() error { return model.sessionReset() }
	}
	res := reset.HandleSlash(input, model.inFlight || turnIsActive(model.frame.Phase), resetFn)
	if !res.Reset {
		return SlashResult{Handled: true, StatusMessage: res.Status}
	}

	model.frame.History = nil
	model.frame.DraftText = ""
	model.frame.LastError = ""
	model.frame.SessionID = ""
	model.sessionID = ""
	model.inFlight = false
	model.ApprovalState = nil
	model.ClarifyState = nil
	model.SecretState = nil
	model.modelPicker = nil

	return SlashResult{Handled: true, StatusMessage: res.Status}
}

func sessionResetSlashKind(input string) string {
	return reset.Kind(input)
}

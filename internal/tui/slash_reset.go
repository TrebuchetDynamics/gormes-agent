package tui

import (
	"fmt"
	"strings"
)

func sessionResetSlashHandler(input string, model *Model) SlashResult {
	kind := sessionResetSlashKind(input)
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: kind + ": TUI unavailable"}
	}
	if model.inFlight || turnIsActive(model.frame.Phase) {
		return SlashResult{Handled: true, StatusMessage: "interrupt the current turn before trying to switch sessions"}
	}
	if model.sessionReset == nil {
		return SlashResult{Handled: true, StatusMessage: kind + ": reset unavailable"}
	}
	if err := model.sessionReset(); err != nil {
		return SlashResult{Handled: true, StatusMessage: fmt.Sprintf("%s: reset failed: %v", kind, err)}
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

	if kind == "new" {
		return SlashResult{Handled: true, StatusMessage: "new session started"}
	}
	return SlashResult{Handled: true, StatusMessage: "session cleared"}
}

func sessionResetSlashKind(input string) string {
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) == 0 {
		return "clear"
	}
	switch strings.ToLower(strings.TrimPrefix(fields[0], "/")) {
	case "new":
		return "new"
	default:
		return "clear"
	}
}

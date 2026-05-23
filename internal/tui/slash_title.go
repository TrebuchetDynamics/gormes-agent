package tui

import (
	"strings"
)

func titleSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "title: TUI unavailable"}
	}
	sessionID := strings.TrimSpace(model.SessionID())
	if sessionID == "" {
		return SlashResult{Handled: true, StatusMessage: "no active session"}
	}
	if model.sessionTitle == nil {
		return SlashResult{Handled: true, StatusMessage: "title: session title unavailable"}
	}

	title, hasTitle := titleSlashArg(input)
	if !hasTitle {
		res, err := model.sessionTitle(sessionID, "")
		if err != nil {
			return SlashResult{Handled: true, StatusMessage: "title: " + err.Error()}
		}
		current := strings.TrimSpace(res.Title)
		if current == "" {
			return SlashResult{Handled: true, StatusMessage: "no title set"}
		}
		return SlashResult{Handled: true, StatusMessage: "title: " + current}
	}
	if title == "" {
		return SlashResult{Handled: true, StatusMessage: "usage: /title <your session title>"}
	}

	res, err := model.sessionTitle(sessionID, title)
	if err != nil {
		return SlashResult{Handled: true, StatusMessage: "title: " + err.Error()}
	}
	next := strings.TrimSpace(res.Title)
	if next == "" {
		next = title
	}
	suffix := ""
	if res.Pending {
		suffix = " (queued while session initializes)"
	}
	return SlashResult{Handled: true, StatusMessage: "session title set: " + next + suffix}
}

func titleSlashArg(input string) (string, bool) {
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) <= 1 {
		return "", false
	}
	return strings.TrimSpace(strings.Join(fields[1:], " ")), true
}

package tui

import (
	"context"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/sessionspage"
)

const sessionResumeTimeout = 5 * time.Second

func sessionsSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "sessions: TUI unavailable"}
	}
	if sessionSlashName(input) == "resume" {
		if arg := sessionSlashArg(input); arg != "" {
			return resumeSlashWithArg(arg, model)
		}
	}
	if model.sessionDirectory == nil {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: "sessions: directory unavailable"}
	}
	entries, err := model.sessionDirectory(sessionsSlashLimit(input))
	if err != nil {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: "sessions: " + err.Error()}
	}
	page, ok := BuildSessionsPage(entries)
	if !ok {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: "no sessions found"}
	}
	model.transientPage = &page
	return SlashResult{Handled: true, StatusMessage: "sessions opened"}
}

func resumeSlashWithArg(query string, model *Model) SlashResult {
	if model.inFlight || turnIsActive(model.frame.Phase) {
		return SlashResult{Handled: true, StatusMessage: "interrupt the current turn before trying to switch sessions"}
	}
	if model.sessionResume == nil {
		return SlashResult{Handled: true, StatusMessage: "resume: session switch unavailable"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), sessionResumeTimeout)
	defer cancel()
	result, err := model.sessionResume(ctx, query)
	if err != nil {
		return SlashResult{Handled: true, StatusMessage: "resume: " + err.Error()}
	}
	sessionID := strings.TrimSpace(result.SessionID)
	if sessionID == "" {
		return SlashResult{Handled: true, StatusMessage: "resume: no session selected"}
	}
	model.sessionID = sessionID
	model.frame.SessionID = sessionID
	model.frame.History = cloneResumeHistory(result.History)
	model.frame.DraftText = ""
	model.frame.LastError = ""
	model.inFlight = false
	model.transientPage = nil
	return SlashResult{Handled: true, StatusMessage: resumeSuccessStatus(sessionID, len(model.frame.History))}
}

func sessionSlashName(input string) string {
	return sessionspage.SlashName(input)
}

func sessionSlashArg(input string) string {
	return sessionspage.SlashArg(input)
}

func resumeSuccessStatus(sessionID string, messages int) string {
	return sessionspage.ResumeSuccessStatus(sessionID, messages)
}

func cloneResumeHistory(in []llm.Message) []llm.Message {
	return sessionspage.CloneResumeHistory(in)
}

func sessionsSlashLimit(input string) int {
	return sessionspage.Limit(input)
}

func BuildSessionsPage(entries []SessionDirectoryEntry) (TransientPageState, bool) {
	return sessionspage.Build(entries)
}

func messageCountLabel(count int) string {
	return sessionspage.MessageCountLabel(count)
}

func sessionDirectoryTimeLabel(ts int64) string {
	return sessionspage.TimeLabel(ts)
}

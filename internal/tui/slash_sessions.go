package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
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
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) == 0 {
		return ""
	}
	return strings.ToLower(strings.TrimPrefix(fields[0], "/"))
}

func sessionSlashArg(input string) string {
	trimmed := strings.TrimSpace(input)
	fields := strings.Fields(trimmed)
	if len(fields) <= 1 {
		return ""
	}
	idx := strings.Index(trimmed, fields[1])
	if idx < 0 {
		return strings.Join(fields[1:], " ")
	}
	return strings.TrimSpace(trimmed[idx:])
}

func resumeSuccessStatus(sessionID string, messages int) string {
	return fmt.Sprintf("resumed %s (%s)", sessionID, messageCountLabel(messages))
}

func cloneResumeHistory(in []hermes.Message) []hermes.Message {
	if len(in) == 0 {
		return nil
	}
	out := make([]hermes.Message, len(in))
	for i, msg := range in {
		out[i] = msg
		out[i].ContentParts = append([]hermes.MessageContentPart(nil), msg.ContentParts...)
	}
	return out
}

func sessionsSlashLimit(input string) int {
	fields := strings.Fields(strings.TrimSpace(input))
	limit := 20
	if len(fields) > 1 {
		if n, err := strconv.Atoi(fields[1]); err == nil {
			limit = n
		}
	}
	if limit < 1 {
		return 1
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func BuildSessionsPage(entries []SessionDirectoryEntry) (TransientPageState, bool) {
	if len(entries) == 0 {
		return TransientPageState{}, false
	}
	blocks := make([]string, 0, len(entries))
	for i, entry := range entries {
		id := strings.TrimSpace(entry.ID)
		if id == "" {
			id = "(unknown session)"
		}
		title := firstNonEmptyString(strings.TrimSpace(entry.Title), strings.TrimSpace(entry.Preview), id)
		preview := strings.TrimSpace(entry.Preview)
		if preview == "" {
			preview = "(no preview)"
		}
		meta := []string{messageCountLabel(entry.MessageCount)}
		if source := strings.TrimSpace(entry.Source); source != "" {
			meta = append(meta, "source: "+source)
		}
		if when := sessionDirectoryTimeLabel(entry.LastActiveAt); when != "" {
			meta = append(meta, "last active: "+when)
		}
		blocks = append(blocks, fmt.Sprintf("%2d. %s\nID: %s\nPreview: %s\n%s", i+1, title, id, preview, strings.Join(meta, " · ")))
	}
	return TransientPageState{Title: "Sessions", Body: strings.Join(blocks, "\n\n")}, true
}

func messageCountLabel(count int) string {
	if count == 1 {
		return "1 message"
	}
	if count < 0 {
		count = 0
	}
	return fmt.Sprintf("%d messages", count)
}

func sessionDirectoryTimeLabel(ts int64) string {
	if ts <= 0 {
		return ""
	}
	return time.Unix(ts, 0).UTC().Format("2006-01-02 15:04 UTC")
}

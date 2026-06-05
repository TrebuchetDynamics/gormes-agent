package title

import "strings"

// SessionTitleResult is the minimal title adapter response HandleSlash needs.
type SessionTitleResult struct {
	Title   string
	Pending bool
}

// SessionTitleFunc gets or sets a session title. Empty title queries current.
type SessionTitleFunc func(sessionID, title string) (SessionTitleResult, error)

// SlashRequest carries the pure /title slash inputs resolved by the root TUI.
type SlashRequest struct {
	Input     string
	SessionID string
	TitleFunc SessionTitleFunc
}

// SlashResult is the behavior-only result returned to the root slash adapter.
type SlashResult struct {
	StatusMessage string
}

// HandleSlash implements /title status, query, validation, and set behavior.
func HandleSlash(req SlashRequest) SlashResult {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return SlashResult{StatusMessage: "no active session"}
	}
	if req.TitleFunc == nil {
		return SlashResult{StatusMessage: "title: session title unavailable"}
	}

	titleArg, hasTitle := SlashArg(req.Input)
	if !hasTitle {
		res, err := req.TitleFunc(sessionID, "")
		if err != nil {
			return SlashResult{StatusMessage: "title: " + err.Error()}
		}
		current := strings.TrimSpace(res.Title)
		if current == "" {
			return SlashResult{StatusMessage: "no title set"}
		}
		return SlashResult{StatusMessage: "title: " + current}
	}
	if titleArg == "" {
		return SlashResult{StatusMessage: "usage: /title <your session title>"}
	}

	res, err := req.TitleFunc(sessionID, titleArg)
	if err != nil {
		return SlashResult{StatusMessage: "title: " + err.Error()}
	}
	next := strings.TrimSpace(res.Title)
	if next == "" {
		next = titleArg
	}
	suffix := ""
	if res.Pending {
		suffix = " (queued while session initializes)"
	}
	return SlashResult{StatusMessage: "session title set: " + next + suffix}
}

// SlashArg returns the collapsed title argument after /title. The bool reports
// whether the caller supplied any argument token; an all-whitespace command has
// no argument and should query the current title.
func SlashArg(input string) (string, bool) {
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) <= 1 {
		return "", false
	}
	return strings.TrimSpace(strings.Join(fields[1:], " ")), true
}

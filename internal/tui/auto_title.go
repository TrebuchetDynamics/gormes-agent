package tui

import "strings"

// AutoTitleInput captures the post-turn state the TUI needs to decide whether
// to request an auto-titled session label for a clean completed prompt. The
// shape mirrors the inputs Hermes 9662e321's tui_gateway/server.py prompt.submit
// path consumes via maybe_auto_title, but adapted to native Gormes session
// metadata: SessionKey is the operator-supplied per-session lookup token and
// FallbackSessionID is the canonical session.SessionID resolved by the kernel.
type AutoTitleInput struct {
	SessionKey        string
	FallbackSessionID string
	Status            string
	UserText          string
	AssistantText     string
	Interrupted       bool
	HistoryCount      int
}

// AutoTitleRequest is the eligibility-resolved descriptor a later wiring row
// can hand to a title generator. It deliberately carries only the resolved
// session ID and the raw user/assistant text bytes so this helper performs no
// title generation, provider call, DB write, goroutine, or clock lookup.
type AutoTitleRequest struct {
	SessionID     string
	UserText      string
	AssistantText string
	HistoryCount  int
}

// BuildAutoTitleRequest decides whether a completed TUI turn is eligible for
// auto-titling and, if so, returns the resolved request. It returns ok=true
// only when Status=="complete", Interrupted is false, the user and assistant
// texts are non-empty after strings.TrimSpace, and the resolved session ID is
// non-empty. Session resolution prefers a trimmed SessionKey and falls back to
// a trimmed FallbackSessionID. The returned request preserves the original
// UserText and AssistantText bytes (no trimming) so a downstream titler sees
// exactly what the turn produced.
func BuildAutoTitleRequest(in AutoTitleInput) (AutoTitleRequest, bool) {
	if in.Status != "complete" {
		return AutoTitleRequest{}, false
	}
	if in.Interrupted {
		return AutoTitleRequest{}, false
	}
	if strings.TrimSpace(in.UserText) == "" {
		return AutoTitleRequest{}, false
	}
	if strings.TrimSpace(in.AssistantText) == "" {
		return AutoTitleRequest{}, false
	}

	sessionID := strings.TrimSpace(in.SessionKey)
	if sessionID == "" {
		sessionID = strings.TrimSpace(in.FallbackSessionID)
	}
	if sessionID == "" {
		return AutoTitleRequest{}, false
	}

	return AutoTitleRequest{
		SessionID:     sessionID,
		UserText:      in.UserText,
		AssistantText: in.AssistantText,
		HistoryCount:  in.HistoryCount,
	}, true
}

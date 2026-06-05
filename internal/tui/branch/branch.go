package branch

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

const ForkTimeout = 5 * time.Second

// Request is the input to the injected /branch adapter.
type Request struct {
	ParentSessionID string
	Title           string
	HistoryCount    int
	History         []llm.Message
}

// Result is the injected /branch adapter response.
type Result struct {
	SessionID        string
	ParentSessionID  string
	Title            string
	TranscriptCopied int
}

// Func is the injection point for the TUI /branch command.
type Func func(ctx context.Context, req Request) (Result, error)

type SlashResult struct {
	Status string
	Branch Result
	Switch bool
}

func HandleSlash(input string, hasConversation bool, parentSessionID string, historyCount int, history []llm.Message, fork Func) SlashResult {
	if !hasConversation {
		return SlashResult{Status: "branch: no conversation"}
	}
	if fork == nil {
		return SlashResult{Status: "branch: store unavailable"}
	}
	parent := strings.TrimSpace(parentSessionID)
	if parent == "" {
		return SlashResult{Status: "branch: no active session"}
	}

	ctx, cancel := context.WithTimeout(context.Background(), ForkTimeout)
	defer cancel()
	res, err := fork(ctx, Request{
		ParentSessionID: parent,
		Title:           TitleFromInput(input),
		HistoryCount:    historyCount,
		History:         history,
	})
	if err != nil {
		return SlashResult{Status: fmt.Sprintf("branch: fork failed: %v", err)}
	}
	return SlashResult{Status: SuccessStatus(res), Branch: res, Switch: true}
}

func TitleFromInput(input string) string {
	trimmed := strings.TrimSpace(input)
	idx := strings.IndexAny(trimmed, " \t")
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(trimmed[idx+1:])
}

func SuccessStatus(res Result) string {
	if res.Title != "" {
		return fmt.Sprintf("branch: switched to %s (%q, %d turns)", res.SessionID, res.Title, res.TranscriptCopied)
	}
	return fmt.Sprintf("branch: switched to %s (%d turns)", res.SessionID, res.TranscriptCopied)
}

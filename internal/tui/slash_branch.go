package tui

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/branch"
)

// branchForkTimeout caps how long /branch waits on the injected helper.
// Forks are local SQLite + bbolt writes — five seconds is generous; if they
// outrun this budget the TUI surfaces a status line instead of blocking the
// editor on a failed disk.
const branchForkTimeout = branch.ForkTimeout

// BranchRequest is the input to SessionBranchFunc. ParentSessionID is the
// session whose persisted history the caller wants forked into a fresh
// child; Title is the operator-supplied label (empty if /branch was typed
// without a name); HistoryCount is the in-memory render-frame turn count
// the TUI saw at fork time, included so the helper can surface a
// `branch: switched (N turns)` status without re-reading the store.
type BranchRequest = branch.Request

// BranchResult is the helper's response. SessionID is the freshly minted
// child id the TUI must switch to; ParentSessionID echoes the parent so
// callers can audit the helper observed the right one; TranscriptCopied
// reports how many parent turns were duplicated under the child.
type BranchResult = branch.Result

// SessionBranchFunc is the injection point for the TUI /branch command.
// cmd/gormes binds the production implementation that calls
// session.Fork+transcript.ForkTurns; tests wire fakes.
type SessionBranchFunc = branch.Func

// branchSlashHandler implements /branch. The handler MUST consume the input
// (Handled=true) on every error branch so the slash text never falls through
// to kernel.Submit; the parent session is left active on every failure so
// degraded-mode operators retain their existing context.
func branchSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "branch: store unavailable"}
	}
	var fork branch.Func
	if model.sessionBranch != nil {
		fork = func(ctx context.Context, req branch.Request) (branch.Result, error) {
			return model.sessionBranch(ctx, req)
		}
	}
	res := branch.HandleSlash(input, len(model.frame.History) > 0, model.SessionID(), len(model.frame.History), cloneResumeHistory(model.frame.History), fork)
	if !res.Switch {
		return SlashResult{Handled: true, StatusMessage: res.Status}
	}

	model.sessionID = res.Branch.SessionID
	model.frame.SessionID = res.Branch.SessionID
	model.inFlight = false
	model.frame.DraftText = ""
	return SlashResult{Handled: true, StatusMessage: res.Status}
}

func branchTitleFromInput(input string) string {
	return branch.TitleFromInput(input)
}

func branchSuccessStatus(res BranchResult) string {
	return branch.SuccessStatus(res)
}

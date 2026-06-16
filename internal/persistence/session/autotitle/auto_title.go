package autotitle

import (
	"context"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/redaction"
)

// Evidence codes describe the single observable outcome of one
// Perform invocation. Codes are stable strings so callers (gateway,
// CLI, TUI) can render uniform user-visible status without retry loops.
const (
	// CodeComplete means the helper persisted a generated title
	// exactly once for an eligible untitled session.
	CodeComplete = "auto_title_complete"

	// CodeSkippedManual means the session already has a manual
	// (user-set) title; the provider was not called and the title was not
	// overwritten.
	CodeSkippedManual = "auto_title_skipped_manual"

	// CodeSkippedTitled means the session already has a non-manual
	// title (previously auto-generated); the helper does not regenerate.
	CodeSkippedTitled = "auto_title_skipped_titled"

	// CodeSkippedNoTranscript means the helper had no eligible
	// user/assistant turns to generate a title from.
	CodeSkippedNoTranscript = "auto_title_skipped_no_transcript"

	// CodeProviderFailed means the title generator returned an
	// error; no title is persisted and no retry is enqueued in this turn.
	CodeProviderFailed = "title_provider_failed"

	// CodeStoreReadFailed means the store could not be queried
	// for the current title state; no provider call or write occurs.
	CodeStoreReadFailed = "auto_title_store_read_failed"

	// CodeStoreWriteFailed means the generator succeeded but the
	// store rejected the write; the title is reported in evidence but no
	// retry is enqueued in this turn.
	CodeStoreWriteFailed = "auto_title_store_write_failed"

	// CodeBlankResult means the generator returned an empty title;
	// no write occurs and no retry is enqueued in this turn.
	CodeBlankResult = "auto_title_blank_result"

	// CodeMissingSession means Perform was called without
	// a session ID; the provider is not called and no write occurs.
	CodeMissingSession = "auto_title_skipped_missing_session"
)

// Turn is the read-only transcript shape consumed by the title generator.
// Perform treats slice elements as immutable bytes; the helper never
// mutates Role or Content and never reslices the input.
type Turn struct {
	Role    string
	Content string
}

// Store is the narrow persistence boundary used by
// Perform. Implementations report whether a title is already present
// and whether it was set manually so manual overrides short-circuit before
// any provider call.
type Store interface {
	// Title returns the persisted title for sessionID, whether it was set
	// manually, whether a row exists at all, and any read error. A
	// non-nil err disables the auto-title path for this turn.
	Title(sessionID string) (current string, manual bool, ok bool, err error)

	// SetTitle persists title for sessionID. Implementations are expected
	// to clear the manual flag (auto-titles are non-manual) atomically with
	// the title write so a follow-up Title call returns manual=false.
	SetTitle(sessionID, title string) error
}

// Generator is the pure provider boundary. Perform calls it at
// most once per invocation; failures surface as evidence and never retry.
type Generator func(ctx context.Context, transcript []Turn) (string, error)

// Evidence is the single outcome record returned by Perform.
// Code is always populated; Title and Reason are populated when relevant.
type Evidence struct {
	Code   string
	Title  string
	Reason string
}

// Perform persists exactly one generated title for sessionID when
// (a) the store reports no existing title, (b) the transcript has at least
// one usable turn, and (c) the generator returns a non-empty result. Manual
// titles short-circuit before the generator is called. Generator failures
// produce title_provider_failed evidence with no write and no retry. The
// helper performs at most one generator call per invocation, holds no
// goroutines, and never mutates the transcript slice or its elements.
func Perform(
	ctx context.Context,
	store Store,
	gen Generator,
	sessionID string,
	transcript []Turn,
) Evidence {
	if strings.TrimSpace(sessionID) == "" {
		return Evidence{
			Code:   CodeMissingSession,
			Reason: "auto-title skipped: missing session id",
		}
	}
	if store == nil {
		return Evidence{
			Code:   CodeStoreReadFailed,
			Reason: "auto-title skipped: nil session title store",
		}
	}
	if gen == nil {
		return Evidence{
			Code:   CodeProviderFailed,
			Reason: "auto-title skipped: nil title generator",
		}
	}
	if !hasEligibleTurn(transcript) {
		return Evidence{
			Code:   CodeSkippedNoTranscript,
			Reason: "auto-title skipped: no eligible transcript turns",
		}
	}

	current, manual, exists, err := store.Title(sessionID)
	if err != nil {
		return Evidence{
			Code:   CodeStoreReadFailed,
			Reason: "auto-title store read failed: " + autoTitleErrorText(err),
		}
	}
	if exists && strings.TrimSpace(current) != "" {
		if manual {
			return Evidence{
				Code:  CodeSkippedManual,
				Title: current,
			}
		}
		return Evidence{
			Code:  CodeSkippedTitled,
			Title: current,
		}
	}

	title, err := gen(ctx, transcript)
	if err != nil {
		return Evidence{
			Code:   CodeProviderFailed,
			Reason: "title provider failed: " + autoTitleErrorText(err),
		}
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return Evidence{
			Code:   CodeBlankResult,
			Reason: "auto-title skipped: title generator returned blank result",
		}
	}

	if err := store.SetTitle(sessionID, title); err != nil {
		return Evidence{
			Code:   CodeStoreWriteFailed,
			Title:  title,
			Reason: "auto-title store write failed: " + autoTitleErrorText(err),
		}
	}

	return Evidence{
		Code:  CodeComplete,
		Title: title,
	}
}

func autoTitleErrorText(err error) string {
	if err == nil {
		return ""
	}
	msg := redaction.RedactSecrets(err.Error())
	msg = strings.NewReplacer("`", "'", "*", "'", "#", "＃").Replace(msg)
	return strings.Join(strings.Fields(msg), " ")
}

// hasEligibleTurn reports whether transcript contains at least one
// user or assistant turn with non-blank content. Other roles (system, tool)
// are ignored so the generator never receives unrelated content.
func hasEligibleTurn(transcript []Turn) bool {
	for _, turn := range transcript {
		switch turn.Role {
		case "user", "assistant":
			if strings.TrimSpace(turn.Content) != "" {
				return true
			}
		}
	}
	return false
}

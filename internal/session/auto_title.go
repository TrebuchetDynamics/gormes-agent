package session

import (
	"context"
	"strings"
)

// AutoTitleEvidence codes describe the single observable outcome of one
// PerformAutoTitle invocation. Codes are stable strings so callers (gateway,
// CLI, TUI) can render uniform user-visible status without retry loops.
const (
	// AutoTitleCodeComplete means the helper persisted a generated title
	// exactly once for an eligible untitled session.
	AutoTitleCodeComplete = "auto_title_complete"

	// AutoTitleCodeSkippedManual means the session already has a manual
	// (user-set) title; the provider was not called and the title was not
	// overwritten.
	AutoTitleCodeSkippedManual = "auto_title_skipped_manual"

	// AutoTitleCodeSkippedTitled means the session already has a non-manual
	// title (previously auto-generated); the helper does not regenerate.
	AutoTitleCodeSkippedTitled = "auto_title_skipped_titled"

	// AutoTitleCodeSkippedNoTranscript means the helper had no eligible
	// user/assistant turns to generate a title from.
	AutoTitleCodeSkippedNoTranscript = "auto_title_skipped_no_transcript"

	// AutoTitleCodeProviderFailed means the title generator returned an
	// error; no title is persisted and no retry is enqueued in this turn.
	AutoTitleCodeProviderFailed = "title_provider_failed"

	// AutoTitleCodeStoreReadFailed means the store could not be queried
	// for the current title state; no provider call or write occurs.
	AutoTitleCodeStoreReadFailed = "auto_title_store_read_failed"

	// AutoTitleCodeStoreWriteFailed means the generator succeeded but the
	// store rejected the write; the title is reported in evidence but no
	// retry is enqueued in this turn.
	AutoTitleCodeStoreWriteFailed = "auto_title_store_write_failed"

	// AutoTitleCodeBlankResult means the generator returned an empty title;
	// no write occurs and no retry is enqueued in this turn.
	AutoTitleCodeBlankResult = "auto_title_blank_result"

	// AutoTitleCodeMissingSession means PerformAutoTitle was called without
	// a session ID; the provider is not called and no write occurs.
	AutoTitleCodeMissingSession = "auto_title_skipped_missing_session"
)

// TitleTurn is the read-only transcript shape consumed by the title generator.
// PerformAutoTitle treats slice elements as immutable bytes; the helper never
// mutates Role or Content and never reslices the input.
type TitleTurn struct {
	Role    string
	Content string
}

// SessionTitleStore is the narrow persistence boundary used by
// PerformAutoTitle. Implementations report whether a title is already present
// and whether it was set manually so manual overrides short-circuit before
// any provider call.
type SessionTitleStore interface {
	// Title returns the persisted title for sessionID, whether it was set
	// manually, whether a row exists at all, and any read error. A
	// non-nil err disables the auto-title path for this turn.
	Title(sessionID string) (current string, manual bool, ok bool, err error)

	// SetTitle persists title for sessionID. Implementations are expected
	// to clear the manual flag (auto-titles are non-manual) atomically with
	// the title write so a follow-up Title call returns manual=false.
	SetTitle(sessionID, title string) error
}

// TitleGenerator is the pure provider boundary. PerformAutoTitle calls it at
// most once per invocation; failures surface as evidence and never retry.
type TitleGenerator func(ctx context.Context, transcript []TitleTurn) (string, error)

// AutoTitleEvidence is the single outcome record returned by PerformAutoTitle.
// Code is always populated; Title and Reason are populated when relevant.
type AutoTitleEvidence struct {
	Code   string
	Title  string
	Reason string
}

// PerformAutoTitle persists exactly one generated title for sessionID when
// (a) the store reports no existing title, (b) the transcript has at least
// one usable turn, and (c) the generator returns a non-empty result. Manual
// titles short-circuit before the generator is called. Generator failures
// produce title_provider_failed evidence with no write and no retry. The
// helper performs at most one generator call per invocation, holds no
// goroutines, and never mutates the transcript slice or its elements.
func PerformAutoTitle(
	ctx context.Context,
	store SessionTitleStore,
	gen TitleGenerator,
	sessionID string,
	transcript []TitleTurn,
) AutoTitleEvidence {
	if strings.TrimSpace(sessionID) == "" {
		return AutoTitleEvidence{
			Code:   AutoTitleCodeMissingSession,
			Reason: "auto-title skipped: missing session id",
		}
	}
	if store == nil {
		return AutoTitleEvidence{
			Code:   AutoTitleCodeStoreReadFailed,
			Reason: "auto-title skipped: nil session title store",
		}
	}
	if gen == nil {
		return AutoTitleEvidence{
			Code:   AutoTitleCodeProviderFailed,
			Reason: "auto-title skipped: nil title generator",
		}
	}
	if !hasEligibleTitleTurn(transcript) {
		return AutoTitleEvidence{
			Code:   AutoTitleCodeSkippedNoTranscript,
			Reason: "auto-title skipped: no eligible transcript turns",
		}
	}

	current, manual, exists, err := store.Title(sessionID)
	if err != nil {
		return AutoTitleEvidence{
			Code:   AutoTitleCodeStoreReadFailed,
			Reason: "auto-title store read failed: " + err.Error(),
		}
	}
	if exists && strings.TrimSpace(current) != "" {
		if manual {
			return AutoTitleEvidence{
				Code:  AutoTitleCodeSkippedManual,
				Title: current,
			}
		}
		return AutoTitleEvidence{
			Code:  AutoTitleCodeSkippedTitled,
			Title: current,
		}
	}

	title, err := gen(ctx, transcript)
	if err != nil {
		return AutoTitleEvidence{
			Code:   AutoTitleCodeProviderFailed,
			Reason: "title provider failed: " + err.Error(),
		}
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return AutoTitleEvidence{
			Code:   AutoTitleCodeBlankResult,
			Reason: "auto-title skipped: title generator returned blank result",
		}
	}

	if err := store.SetTitle(sessionID, title); err != nil {
		return AutoTitleEvidence{
			Code:   AutoTitleCodeStoreWriteFailed,
			Title:  title,
			Reason: "auto-title store write failed: " + err.Error(),
		}
	}

	return AutoTitleEvidence{
		Code:  AutoTitleCodeComplete,
		Title: title,
	}
}

// hasEligibleTitleTurn reports whether transcript contains at least one
// user or assistant turn with non-blank content. Other roles (system, tool)
// are ignored so the generator never receives unrelated content.
func hasEligibleTitleTurn(transcript []TitleTurn) bool {
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

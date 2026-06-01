package session

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session/autotitle"
)

// AutoTitleEvidence codes describe the single observable outcome of one
// PerformAutoTitle invocation. Codes are stable strings so callers (gateway,
// CLI, TUI) can render uniform user-visible status without retry loops.
const (
	AutoTitleCodeComplete            = autotitle.CodeComplete
	AutoTitleCodeSkippedManual       = autotitle.CodeSkippedManual
	AutoTitleCodeSkippedTitled       = autotitle.CodeSkippedTitled
	AutoTitleCodeSkippedNoTranscript = autotitle.CodeSkippedNoTranscript
	AutoTitleCodeProviderFailed      = autotitle.CodeProviderFailed
	AutoTitleCodeStoreReadFailed     = autotitle.CodeStoreReadFailed
	AutoTitleCodeStoreWriteFailed    = autotitle.CodeStoreWriteFailed
	AutoTitleCodeBlankResult         = autotitle.CodeBlankResult
	AutoTitleCodeMissingSession      = autotitle.CodeMissingSession
)

type TitleTurn = autotitle.Turn
type SessionTitleStore = autotitle.Store
type TitleGenerator = autotitle.Generator
type AutoTitleEvidence = autotitle.Evidence

func PerformAutoTitle(
	ctx context.Context,
	store SessionTitleStore,
	gen TitleGenerator,
	sessionID string,
	transcript []TitleTurn,
) AutoTitleEvidence {
	return autotitle.Perform(ctx, store, gen, sessionID, transcript)
}

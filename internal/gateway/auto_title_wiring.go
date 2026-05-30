package gateway

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/autotitle"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
)

// AutoTitleAuxiliarySink receives AutoTitleEvidence for non-complete outcomes
// (provider failures, blank results, store errors). Implementations must not
// block or panic; the caller recovers and logs panics.
type AutoTitleAuxiliarySink = autotitle.AuxiliarySink

// runAutoTitle builds a two-turn TitleTurn transcript from lastUserText and
// finalAssistantText, calls session.PerformAutoTitle once, and routes the
// resulting AutoTitleEvidence through the auxiliary sink for non-complete
// outcomes. The call is synchronous and bounded; no goroutines are started.
//
// sessionID must be non-empty; an empty sessionID yields an
// AutoTitleCodeMissingSession outcome that is still routed through the sink.
func runAutoTitle(
	ctx context.Context,
	store session.SessionTitleStore,
	gen session.TitleGenerator,
	sessionID string,
	lastUserText string,
	finalAssistantText string,
	sink AutoTitleAuxiliarySink,
) {
	autotitle.Run(ctx, store, gen, sessionID, lastUserText, finalAssistantText, sink)
}

// maybeRunAutoTitle is the gateway entry point for auto-title generation. It
// is called from dispatchFrame after a PhaseIdle frame is delivered. It:
//   - skips silently when TitleStore is nil
//   - wraps cfg.TitleModel as a session.TitleGenerator (nil model is allowed;
//     PerformAutoTitle surfaces AutoTitleCodeProviderFailed via sink)
//   - extracts the final assistant text from the frame History
//   - delegates to runAutoTitle with the configured AuxiliaryFailureSink
func (m *Manager) maybeRunAutoTitle(ctx context.Context, f kernel.RenderFrame, sessionID, lastUserText string) {
	store := m.cfg.TitleStore
	if store == nil {
		return
	}

	var gen session.TitleGenerator
	if m.cfg.TitleModel != nil {
		titleModel := m.cfg.TitleModel
		gen = func(ctx context.Context, transcript []session.TitleTurn) (string, error) {
			return titleModelToGenerator(ctx, titleModel, transcript)
		}
	}

	finalAssistantText := lastAssistantText(f)
	runAutoTitle(ctx, store, gen, sessionID, lastUserText, finalAssistantText, m.cfg.AuxiliaryFailureSink)
}

// titleModelToGenerator adapts a llm.TitleModelFunc to the session.TitleGenerator
// signature expected by PerformAutoTitle. It maps session.TitleTurn slices to
// llm.TitleModelRequest messages so the title model boundary is
// platform-independent.
func titleModelToGenerator(ctx context.Context, fn llm.TitleModelFunc, transcript []session.TitleTurn) (string, error) {
	return autotitle.TitleModelToGenerator(ctx, fn, transcript)
}

// lastAssistantText extracts the Content of the last assistant-role message in
// the frame History. Returns empty string when no assistant message is found.
func lastAssistantText(f kernel.RenderFrame) string {
	return autotitle.LastAssistantText(f)
}

package gateway

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
)

// AutoTitleAuxiliarySink receives AutoTitleEvidence for non-complete outcomes
// (provider failures, blank results, store errors). Implementations must not
// block or panic; the caller recovers panics and discarded.
type AutoTitleAuxiliarySink func(ctx context.Context, ev session.AutoTitleEvidence)

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
	transcript := []session.TitleTurn{
		{Role: "user", Content: lastUserText},
		{Role: "assistant", Content: finalAssistantText},
	}
	ev := session.PerformAutoTitle(ctx, store, gen, sessionID, transcript)
	if ev.Code == session.AutoTitleCodeComplete {
		return
	}
	if sink == nil {
		return
	}
	func() {
		defer func() { recover() }() //nolint:errcheck // sink panics are discarded
		sink(ctx, ev)
	}()
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

// titleModelToGenerator adapts a hermes.TitleModelFunc to the session.TitleGenerator
// signature expected by PerformAutoTitle. It maps session.TitleTurn slices to
// hermes.TitleModelRequest messages so the title model boundary is
// platform-independent.
func titleModelToGenerator(ctx context.Context, fn hermes.TitleModelFunc, transcript []session.TitleTurn) (string, error) {
	msgs := make([]hermes.TitleModelMessage, 0, len(transcript))
	for _, t := range transcript {
		msgs = append(msgs, hermes.TitleModelMessage{Role: t.Role, Content: t.Content})
	}
	return fn(ctx, hermes.TitleModelRequest{
		Messages:    msgs,
		MaxTokens:   500,
		Temperature: 0.3,
	})
}

// lastAssistantText extracts the Content of the last assistant-role message in
// the frame History. Returns empty string when no assistant message is found.
func lastAssistantText(f kernel.RenderFrame) string {
	for i := len(f.History) - 1; i >= 0; i-- {
		if f.History[i].Role == "assistant" {
			return f.History[i].Content
		}
	}
	return ""
}

package autotitle

import (
	"context"
	"log/slog"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/renderframe"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
)

// AuxiliarySink receives AutoTitleEvidence for non-complete outcomes (provider
// failures, blank results, store errors). Implementations must not block or
// panic; the caller recovers and logs panics.
type AuxiliarySink func(ctx context.Context, ev session.AutoTitleEvidence)

// Run builds a two-turn TitleTurn transcript from lastUserText and
// finalAssistantText, calls session.PerformAutoTitle once, and routes the
// resulting AutoTitleEvidence through the auxiliary sink for non-complete
// outcomes. The call is synchronous and bounded; no goroutines are started.
//
// sessionID must be non-empty; an empty sessionID yields an
// AutoTitleCodeMissingSession outcome that is still routed through the sink.
func Run(
	ctx context.Context,
	store session.SessionTitleStore,
	gen session.TitleGenerator,
	sessionID string,
	lastUserText string,
	finalAssistantText string,
	sink AuxiliarySink,
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
		defer func() {
			if r := recover(); r != nil {
				slog.Error("auto_title_sink_panic", "panic", r)
			}
		}()
		sink(ctx, ev)
	}()
}

// TitleModelToGenerator adapts a llm.TitleModelFunc to the
// session.TitleGenerator signature expected by PerformAutoTitle. It maps
// session.TitleTurn slices to llm.TitleModelRequest messages so the title model
// boundary is platform-independent.
func TitleModelToGenerator(ctx context.Context, fn llm.TitleModelFunc, transcript []session.TitleTurn) (string, error) {
	msgs := make([]llm.TitleModelMessage, 0, len(transcript))
	for _, t := range transcript {
		msgs = append(msgs, llm.TitleModelMessage{Role: t.Role, Content: t.Content})
	}
	return fn(ctx, llm.TitleModelRequest{
		Messages:    msgs,
		MaxTokens:   500,
		Temperature: 0.3,
	})
}

// LastAssistantText extracts the Content of the last assistant-role message in
// the frame History. Returns empty string when no assistant message is found.
func LastAssistantText(f kernel.RenderFrame) string {
	return renderframe.LastAssistantText(f)
}

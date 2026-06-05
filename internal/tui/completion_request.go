package tui

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/completion"

type TUICompletionMethod = completion.Method

const (
	TUICompletionPath  TUICompletionMethod = completion.Path
	TUICompletionSlash TUICompletionMethod = completion.Slash
)

// TUICompletionRequest is the native Bubble Tea equivalent of Hermes Ink's
// completionRequestForInput return shape. Text is set for slash completion;
// Word is set for path completion.
type TUICompletionRequest = completion.Request

// CompletionRequestForInput classifies editor text for the generic completion
// surface. It mirrors current Hermes Ink behavior: real slash commands use
// slash completion, but slash-looking absolute paths stay path completions.
func CompletionRequestForInput(input string) (TUICompletionRequest, bool) {
	return completion.RequestForInput(input)
}

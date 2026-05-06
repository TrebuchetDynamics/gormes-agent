package tui

import "regexp"

type TUICompletionMethod string

const (
	TUICompletionPath  TUICompletionMethod = "complete.path"
	TUICompletionSlash TUICompletionMethod = "complete.slash"
)

// TUICompletionRequest is the native Bubble Tea equivalent of Hermes Ink's
// completionRequestForInput return shape. Text is set for slash completion;
// Word is set for path completion.
type TUICompletionRequest struct {
	Method      TUICompletionMethod
	Text        string
	Word        string
	ReplaceFrom int
}

var (
	tuiSlashCommandRE      = regexp.MustCompile(`^/[^\s/]*(?:\s|$)`)
	tuiPickerSlashRE       = regexp.MustCompile(`^/(?:model|provider)(?:\s|$)`)
	tuiCompletionPathWordR = regexp.MustCompile("((?:[\"']?(?:[A-Za-z]:[\\\\/]|\\.{1,2}/|~/|/|@|[^\"'`\\s]+/))[^\\s]*)$")
)

// CompletionRequestForInput classifies editor text for the generic completion
// surface. It mirrors current Hermes Ink behavior: real slash commands use
// slash completion, but slash-looking absolute paths stay path completions.
func CompletionRequestForInput(input string) (TUICompletionRequest, bool) {
	isSlashCommand := tuiSlashCommandRE.MatchString(input)
	pathWord := ""
	if !isSlashCommand {
		if match := tuiCompletionPathWordR.FindStringSubmatch(input); len(match) > 1 {
			pathWord = match[1]
		}
	}

	if !isSlashCommand && pathWord == "" {
		return TUICompletionRequest{}, false
	}

	if isSlashCommand && tuiPickerSlashRE.MatchString(input) {
		return TUICompletionRequest{}, false
	}

	if isSlashCommand {
		return TUICompletionRequest{
			Method:      TUICompletionSlash,
			Text:        input,
			ReplaceFrom: 1,
		}, true
	}

	return TUICompletionRequest{
		Method:      TUICompletionPath,
		Word:        pathWord,
		ReplaceFrom: len(input) - len(pathWord),
	}, true
}

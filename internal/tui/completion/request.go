package completion

import "regexp"

type Method string

const (
	Path  Method = "complete.path"
	Slash Method = "complete.slash"
)

// Request is the native Bubble Tea equivalent of Hermes Ink's
// completionRequestForInput return shape. Text is set for slash completion;
// Word is set for path completion.
type Request struct {
	Method      Method
	Text        string
	Word        string
	ReplaceFrom int
}

var (
	slashCommandRE      = regexp.MustCompile(`^/[^\s/]*(?:\s|$)`)
	pickerSlashRE       = regexp.MustCompile(`^/(?:model|provider)(?:\s|$)`)
	completionPathWordR = regexp.MustCompile("((?:[\"']?(?:[A-Za-z]:[\\\\/]|\\.{1,2}/|~/|/|@|[^\"'`\\s]+/))[^\\s]*)$")
)

// RequestForInput classifies editor text for the generic completion
// surface. It mirrors current Hermes Ink behavior: real slash commands use
// slash completion, but slash-looking absolute paths stay path completions.
func RequestForInput(input string) (Request, bool) {
	isSlashCommand := slashCommandRE.MatchString(input)
	pathWord := ""
	if !isSlashCommand {
		if match := completionPathWordR.FindStringSubmatch(input); len(match) > 1 {
			pathWord = match[1]
		}
	}

	if !isSlashCommand && pathWord == "" {
		return Request{}, false
	}

	if isSlashCommand && pickerSlashRE.MatchString(input) {
		return Request{}, false
	}

	if isSlashCommand {
		return Request{
			Method:      Slash,
			Text:        input,
			ReplaceFrom: 1,
		}, true
	}

	return Request{
		Method:      Path,
		Word:        pathWord,
		ReplaceFrom: len(input) - len(pathWord),
	}, true
}

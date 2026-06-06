package slashcompletion

import "strings"

type completionRequestKind int

const (
	completionRequestCommand completionRequestKind = iota
	completionRequestSubcommand
)

type completionRequest struct {
	kind          completionRequestKind
	commandPrefix completionPrefix
	base          string
	subPrefix     completionPrefix
}

type completionPrefix string

func newCompletionPrefix(raw string) completionPrefix {
	return completionPrefix(completionKey(raw))
}

func newSubcommandPrefix(raw string) completionPrefix {
	return completionPrefix(strings.ToLower(strings.TrimSpace(raw)))
}

func (p completionPrefix) string() string {
	return string(p)
}

func (p completionPrefix) matches(name string) bool {
	return strings.HasPrefix(completionKey(name), p.string())
}

func (r completionRequest) commandOnly() bool {
	return r.kind == completionRequestCommand
}

func (r completionRequest) subcommandOnly() bool {
	return r.kind == completionRequestSubcommand
}

func parseCompletionRequest(input string) (completionRequest, bool) {
	if !strings.HasPrefix(input, "/") {
		return completionRequest{}, false
	}
	sep := strings.IndexFunc(input, func(r rune) bool { return r == ' ' || r == '\t' })
	if sep < 0 {
		return completionRequest{kind: completionRequestCommand, commandPrefix: newCompletionPrefix(input)}, true
	}
	base := completionKey(input[:sep])
	if base == "" {
		return completionRequest{}, false
	}
	subText := strings.TrimLeft(input[sep+1:], " \t")
	if strings.ContainsAny(subText, " \t") {
		return completionRequest{}, false
	}
	return completionRequest{
		kind:      completionRequestSubcommand,
		base:      "/" + base,
		subPrefix: newSubcommandPrefix(subText),
	}, true
}

func completionNameMatches(name string, prefix completionPrefix) bool {
	return prefix.matches(name)
}

func completionName(name string) string {
	trimmed := strings.TrimSpace(name)
	trimmed = strings.TrimLeft(trimmed, "/")
	return strings.TrimSpace(trimmed)
}

func completionKey(name string) string {
	return strings.ToLower(completionName(name))
}

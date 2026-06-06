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

type completionInputParts struct {
	command           string
	subword           string
	hasSubcommandSlot bool
	hasArgs           bool
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
	parts, ok := splitCompletionInput(input)
	if !ok {
		return completionRequest{}, false
	}
	if parts.commandOnly() {
		return completionRequest{kind: completionRequestCommand, commandPrefix: newCompletionPrefix(parts.command)}, true
	}
	if !parts.subcommandCandidate() {
		return completionRequest{}, false
	}
	base := completionKey(parts.command)
	if base == "" {
		return completionRequest{}, false
	}
	return completionRequest{
		kind:      completionRequestSubcommand,
		base:      "/" + base,
		subPrefix: newSubcommandPrefix(parts.subword),
	}, true
}

func splitCompletionInput(input string) (completionInputParts, bool) {
	if !strings.HasPrefix(input, "/") {
		return completionInputParts{}, false
	}
	sep := strings.IndexFunc(input, isCompletionWhitespace)
	if sep < 0 {
		return completionInputParts{command: input}, true
	}
	subText := strings.TrimLeft(input[sep+1:], " \t")
	nextSep := strings.IndexFunc(subText, isCompletionWhitespace)
	if nextSep >= 0 {
		return completionInputParts{command: input[:sep], subword: subText[:nextSep], hasSubcommandSlot: true, hasArgs: true}, true
	}
	return completionInputParts{command: input[:sep], subword: subText, hasSubcommandSlot: true}, true
}

func isCompletionWhitespace(r rune) bool {
	return r == ' ' || r == '\t'
}

func (p completionInputParts) commandOnly() bool {
	return !p.hasSubcommandSlot
}

func (p completionInputParts) subcommandCandidate() bool {
	return p.hasSubcommandSlot && !p.hasArgs
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

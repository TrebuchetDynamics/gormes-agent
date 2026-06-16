package slashcompletion

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/slashcompletion/normalization"
)

const completionWhitespaceChars = " \t"

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

type completionPrefix = normalization.Prefix

type completionIdentity = normalization.Identity

func newCompletionIdentity(raw string) completionIdentity {
	return normalization.NewIdentity(raw)
}

func newCompletionPrefix(raw string) completionPrefix {
	return normalization.NewCommandPrefix(raw)
}

func newSubcommandPrefix(raw string) completionPrefix {
	return normalization.NewSubcommandPrefix(raw)
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
	sep := indexCompletionWhitespace(input)
	if sep < 0 {
		return completionInputParts{command: input}, true
	}
	subText := trimCompletionWhitespaceLeft(input[sep+1:])
	nextSep := indexCompletionWhitespace(subText)
	if nextSep >= 0 {
		return completionInputParts{command: input[:sep], subword: subText[:nextSep], hasSubcommandSlot: true, hasArgs: true}, true
	}
	return completionInputParts{command: input[:sep], subword: subText, hasSubcommandSlot: true}, true
}

func indexCompletionWhitespace(input string) int {
	return strings.IndexFunc(input, isCompletionWhitespace)
}

func trimCompletionWhitespaceLeft(input string) string {
	return strings.TrimLeft(input, completionWhitespaceChars)
}

func containsCompletionWhitespace(input string) bool {
	return strings.ContainsAny(input, completionWhitespaceChars)
}

func isCompletionWhitespace(r rune) bool {
	return strings.ContainsRune(completionWhitespaceChars, r)
}

func (p completionInputParts) commandOnly() bool {
	return !p.hasSubcommandSlot
}

func (p completionInputParts) subcommandCandidate() bool {
	return p.hasSubcommandSlot && !p.hasArgs
}

func completionNameMatches(name string, prefix completionPrefix) bool {
	return prefix.Matches(name)
}

func completionName(name string) string {
	return normalization.Name(name)
}

func trimCompletionSlashPrefix(name string) string {
	return normalization.TrimSlashPrefix(name)
}

func completionKey(name string) string {
	return normalization.Key(name)
}

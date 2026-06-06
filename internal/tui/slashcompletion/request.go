package slashcompletion

import "strings"

type completionRequestKind int

const (
	completionRequestCommand completionRequestKind = iota
	completionRequestSubcommand
)

type completionRequest struct {
	kind          completionRequestKind
	commandPrefix string
	base          string
	subPrefix     string
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
		return completionRequest{kind: completionRequestCommand, commandPrefix: completionKey(input)}, true
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
		subPrefix: strings.ToLower(subText),
	}, true
}

func completionNameMatches(name, prefix string) bool {
	return strings.HasPrefix(completionKey(name), prefix)
}

func completionKey(name string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(name), "/"))
}

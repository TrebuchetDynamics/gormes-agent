package slashcompletion

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

type acceptedTextPlan struct {
	Text    string
	Changed bool
	Reason  acceptedTextReason
}

type acceptedTextReason string

const (
	acceptedTextReasonEmptyCompletion acceptedTextReason = "empty-completion"
	acceptedTextReasonSubcommand      acceptedTextReason = "subcommand"
	acceptedTextReasonExactRejected   acceptedTextReason = "exact-rejected"
	acceptedTextReasonCommand         acceptedTextReason = "command"
)

func AcceptedText(input string, completion Completion, acceptExact bool) (string, bool) {
	plan := planAcceptedText(input, completion, acceptExact)
	return plan.Text, plan.Changed
}

func planAcceptedText(input string, completion Completion, acceptExact bool) acceptedTextPlan {
	name := strings.TrimSpace(strings.TrimPrefix(completion.Name, "/"))
	if name == "" {
		return acceptedTextPlan{Text: input, Reason: acceptedTextReasonEmptyCompletion}
	}
	if base, ok := subcommandBase(input); ok {
		next := base + " " + name
		return acceptedTextPlan{Text: next, Changed: next != input, Reason: acceptedTextReasonSubcommand}
	}

	next := "/" + name
	exact := strings.TrimSpace(input) == next
	if exact && !acceptExact {
		return acceptedTextPlan{Text: input, Reason: acceptedTextReasonExactRejected}
	}
	if shouldAppendSpace(completion) && (acceptExact || !exact) {
		next += " "
	}
	return acceptedTextPlan{Text: next, Changed: next != input, Reason: acceptedTextReasonCommand}
}

func subcommandBase(input string) (string, bool) {
	base, _, ok := splitSubcommandInput(input)
	if !ok {
		return "", false
	}
	policy, ok := cli.ResolveCommandPolicy(base)
	if !ok || len(policy.Subcommands) == 0 {
		return "", false
	}
	return base, true
}

func splitSubcommandInput(input string) (base string, subText string, ok bool) {
	req, ok := parseCompletionRequest(input)
	if !ok || !req.subcommandOnly() {
		return "", "", false
	}
	return req.base, req.subPrefix, true
}

func shouldAppendSpace(completion Completion) bool {
	name := strings.TrimSpace(strings.TrimPrefix(completion.Name, "/"))
	if name == "" {
		return false
	}
	if _, ok := noTrailingSpaceCommands[name]; ok {
		return false
	}
	if strings.TrimSpace(completion.ArgumentHint) != "" {
		return true
	}
	policy, ok := cli.ResolveCommandPolicy(name)
	if !ok {
		return false
	}
	if _, ok := noTrailingSpaceCommands[policy.Name]; ok {
		return false
	}
	if len(policy.Subcommands) > 0 {
		return true
	}
	_, ok = argumentCommandNames[policy.Name]
	return ok
}

var noTrailingSpaceCommands = map[string]struct{}{
	"model":       {},
	"personality": {},
	"provider":    {},
	"skin":        {},
}

var argumentCommandNames = map[string]struct{}{
	"approve":    {},
	"background": {},
	"branch":     {},
	"commands":   {},
	"compress":   {},
	"copy":       {},
	"cron":       {},
	"curator":    {},
	"goal":       {},
	"image":      {},
	"insights":   {},
	"new":        {},
	"platform":   {},
	"queue":      {},
	"quit":       {},
	"resume":     {},
	"rollback":   {},
	"sessions":   {},
	"snapshot":   {},
	"steer":      {},
	"subgoal":    {},
	"title":      {},
	"topic":      {},
	"tools":      {},
}

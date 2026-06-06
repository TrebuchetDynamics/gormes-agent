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
	accepted := newAcceptedCompletion(completion)
	if accepted.empty() {
		return acceptedTextPlan{Text: input, Reason: acceptedTextReasonEmptyCompletion}
	}
	if base, ok := subcommandBase(input); ok {
		next := base + " " + accepted.name
		return acceptedTextPlan{Text: next, Changed: next != input, Reason: acceptedTextReasonSubcommand}
	}

	next := "/" + accepted.name
	exact := strings.TrimSpace(input) == next
	if exact && !acceptExact {
		return acceptedTextPlan{Text: input, Reason: acceptedTextReasonExactRejected}
	}
	if accepted.shouldAppendSpace() && (acceptExact || !exact) {
		next += " "
	}
	return acceptedTextPlan{Text: next, Changed: next != input, Reason: acceptedTextReasonCommand}
}

func subcommandBase(input string) (string, bool) {
	flow, ok := resolveSubcommandFlow(input)
	if !ok {
		return "", false
	}
	return flow.Base, true
}

type acceptedCompletion struct {
	name         string
	key          string
	argumentHint string
}

func newAcceptedCompletion(completion Completion) acceptedCompletion {
	name := completionName(completion.Name)
	return acceptedCompletion{
		name:         name,
		key:          completionKey(name),
		argumentHint: strings.TrimSpace(completion.ArgumentHint),
	}
}

func (c acceptedCompletion) empty() bool {
	return c.name == ""
}

func (c acceptedCompletion) shouldAppendSpace() bool {
	if c.empty() {
		return false
	}
	if _, ok := noTrailingSpaceCommands[c.key]; ok {
		return false
	}
	if c.argumentHint != "" {
		return true
	}
	policy, ok := cli.ResolveCommandPolicy(c.key)
	if !ok {
		return false
	}
	policyKey := completionKey(policy.Name)
	if _, ok := noTrailingSpaceCommands[policyKey]; ok {
		return false
	}
	if len(policy.Subcommands) > 0 {
		return true
	}
	_, ok = argumentCommandNames[policyKey]
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
	"title":      {},
	"topic":      {},
	"tools":      {},
}

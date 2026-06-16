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
	acceptedTextReasonEmptyCompletion  acceptedTextReason = "empty-completion"
	acceptedTextReasonSubcommand       acceptedTextReason = "subcommand"
	acceptedTextReasonExactRejected    acceptedTextReason = "exact-rejected"
	acceptedTextReasonUnsupportedInput acceptedTextReason = "unsupported-input"
	acceptedTextReasonCommand          acceptedTextReason = "command"
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
	if inputHasCompletionArguments(input) {
		return acceptedTextPlan{Text: input, Reason: acceptedTextReasonUnsupportedInput}
	}
	if flow, ok := resolveSubcommandFlow(input); ok {
		if !acceptedSubcommandCandidate(flow, accepted) {
			return acceptedTextPlan{Text: input, Reason: acceptedTextReasonUnsupportedInput}
		}
		return planAcceptedSubcommandText(input, flow, accepted, acceptExact)
	}
	if !acceptedCommandCandidate(input, accepted) {
		return acceptedTextPlan{Text: input, Reason: acceptedTextReasonUnsupportedInput}
	}

	next := "/" + accepted.name
	decision := decideAcceptedCommandText(next, newCompletionPrefix(input).String() == accepted.key, accepted.shouldAppendSpace(), acceptExact)
	if decision.exact && !acceptExact {
		return acceptedTextPlan{Text: input, Reason: acceptedTextReasonExactRejected}
	}
	return acceptedTextPlan{Text: decision.text, Changed: decision.text != input, Reason: acceptedTextReasonCommand}
}

type acceptedCommandTextDecision struct {
	text  string
	exact bool
}

func decideAcceptedCommandText(normalized string, exact, appendSpace, acceptExact bool) acceptedCommandTextDecision {
	decision := acceptedCommandTextDecision{
		text:  normalized,
		exact: exact,
	}
	if appendSpace && (acceptExact || !decision.exact) {
		decision.text += " "
	}
	return decision
}

func planAcceptedSubcommandText(input string, flow subcommandFlow, accepted acceptedCompletion, acceptExact bool) acceptedTextPlan {
	next := flow.Base + " " + accepted.name
	decision := decideAcceptedSubcommandText(next, flow.Prefix.String() == accepted.key, acceptExact)
	if decision.exact && !acceptExact {
		return acceptedTextPlan{Text: input, Reason: acceptedTextReasonExactRejected}
	}
	return acceptedTextPlan{Text: decision.text, Changed: decision.text != input, Reason: acceptedTextReasonSubcommand}
}

type acceptedSubcommandTextDecision struct {
	text  string
	exact bool
}

func decideAcceptedSubcommandText(normalized string, exact, acceptExact bool) acceptedSubcommandTextDecision {
	decision := acceptedSubcommandTextDecision{
		text:  normalized,
		exact: exact,
	}
	if acceptExact && decision.exact {
		decision.text += " "
	}
	return decision
}

func inputHasCompletionArguments(input string) bool {
	parts, ok := splitCompletionInput(input)
	return ok && parts.hasArgs
}

func acceptedCommandCandidate(input string, accepted acceptedCompletion) bool {
	req, ok := parseCompletionRequest(input)
	return ok && req.commandOnly() && req.commandPrefix.Matches(accepted.name)
}

func acceptedSubcommandCandidate(flow subcommandFlow, accepted acceptedCompletion) bool {
	for _, candidate := range matchingSubcommandCandidates(flow.Subcommands, flow.Prefix) {
		if candidate.key == accepted.key {
			return true
		}
	}
	return false
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
	return decideAcceptedCompletionSpacing(c).AppendSpace
}

type acceptedCompletionSpacingDecision struct {
	AppendSpace bool
	Reason      acceptedCompletionSpacingReason
}

type acceptedCompletionSpacingReason string

const (
	acceptedCompletionSpacingEmpty             acceptedCompletionSpacingReason = "empty"
	acceptedCompletionSpacingNoTrailingSpace   acceptedCompletionSpacingReason = "no-trailing-space"
	acceptedCompletionSpacingPolicySubcommands acceptedCompletionSpacingReason = "policy-subcommands"
	acceptedCompletionSpacingPolicyArgument    acceptedCompletionSpacingReason = "policy-argument"
	acceptedCompletionSpacingArgumentHint      acceptedCompletionSpacingReason = "argument-hint"
	acceptedCompletionSpacingUnknown           acceptedCompletionSpacingReason = "unknown"
)

func decideAcceptedCompletionSpacing(c acceptedCompletion) acceptedCompletionSpacingDecision {
	if c.empty() {
		return acceptedCompletionSpacingDecision{Reason: acceptedCompletionSpacingEmpty}
	}
	if _, ok := noTrailingSpaceCommands[c.key]; ok {
		return acceptedCompletionSpacingDecision{Reason: acceptedCompletionSpacingNoTrailingSpace}
	}
	if policy, ok := cli.ResolveCommandPolicy(c.key); ok {
		policyKey := completionKey(policy.Name)
		if _, ok := noTrailingSpaceCommands[policyKey]; ok {
			return acceptedCompletionSpacingDecision{Reason: acceptedCompletionSpacingNoTrailingSpace}
		}
		if len(policy.Subcommands) > 0 {
			return acceptedCompletionSpacingDecision{AppendSpace: true, Reason: acceptedCompletionSpacingPolicySubcommands}
		}
		if _, ok := argumentCommandNames[policyKey]; ok {
			return acceptedCompletionSpacingDecision{AppendSpace: true, Reason: acceptedCompletionSpacingPolicyArgument}
		}
		return acceptedCompletionSpacingDecision{Reason: acceptedCompletionSpacingUnknown}
	}
	if c.argumentHint != "" {
		return acceptedCompletionSpacingDecision{AppendSpace: true, Reason: acceptedCompletionSpacingArgumentHint}
	}
	return acceptedCompletionSpacingDecision{Reason: acceptedCompletionSpacingUnknown}
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

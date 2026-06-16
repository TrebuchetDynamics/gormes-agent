package slashcompletion

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"

type subcommandFlow struct {
	Base        string
	Prefix      completionPrefix
	Subcommands []string
}

func resolveSubcommandFlow(input string) (subcommandFlow, bool) {
	req, ok := parseCompletionRequest(input)
	if !ok || !req.subcommandOnly() {
		return subcommandFlow{}, false
	}
	policy, ok := cli.ResolveCommandPolicy(req.base)
	if !ok || len(policy.Subcommands) == 0 {
		return subcommandFlow{}, false
	}
	return subcommandFlow{Base: req.base, Prefix: req.subPrefix, Subcommands: policy.Subcommands}, true
}

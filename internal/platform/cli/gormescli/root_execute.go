package gormescli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

// RootExecutionOptions configures process-level argument handling that wraps
// the Cobra root command: compatibility guidance for removed shortcuts,
// deterministic typo hints, and JSON input errors for unknown subcommands.
type RootExecutionOptions struct {
	BuildProvenance  func() BuildProvenance
	ExitCodeError    func(int, error) error
	HandledExitError func(error) bool
	TypoSuggestion   func([]string) (string, bool)
}

// ExecuteRootCommand applies Gormes' pre/post-Cobra root execution guards and
// then executes root with args. It preserves the public CLI compatibility
// contract for removed Hermes/Gormes shortcuts while keeping cmd/gormes as a
// thin executable facade.
func ExecuteRootCommand(root *cobra.Command, args []string, opts RootExecutionOptions) error {
	args = CoalesceSessionNameArgs(args)
	if suggestion, ok := RemovedRootFlagSuggestion(args); ok {
		fmt.Fprintf(root.ErrOrStderr(), "%s\n", suggestion)
		return rootExecutionExitError(opts, 2, fmt.Errorf("%s", suggestion))
	}
	if suggestion, ok := rootExecutionTypoSuggestion(opts, args); ok {
		fmt.Fprintf(root.ErrOrStderr(), "unknown command %q for %q\n%s\n", args[0], root.CommandPath(), suggestion)
		return rootExecutionExitError(opts, 1, fmt.Errorf("unknown command %q for %q; %s", args[0], root.CommandPath(), suggestion))
	}
	if len(args) > 0 {
		root.SetArgs(args)
	}
	err := root.Execute()
	// Catch cobra's `Find()`/`findSuggestions` short-circuit:
	// `gormes config gat --json` produces an `unknown command "gat"
	// for "gormes config"; did you mean "get"?` error returned
	// directly from Find(), bypassing the parent's RunE guard installed by
	// InstallParentUnknownSubcommandGuards. When --json is in args,
	// escalate that error into a structured JSON document on stdout so fleet
	// automation sees the same `{build, action: "unknown_subcommand", error}`
	// shape it gets for the no-suggestion case.
	//
	// Skip when the error is already a handled exit-code error — that means
	// some inner RunE (mcp parent guard, the recursive parent guards) already
	// emitted a JSON document; double-emitting would corrupt stdout.
	if err != nil && ArgsIncludeJSONFlag(args) && !rootExecutionHandledExitError(opts, err) {
		if isCobraUnknownCommandError(err) {
			return emitRootExecutionJSONInputError(root, opts, "unknown_subcommand", err.Error())
		}
		if isCobraUnknownJSONFlagError(err) {
			return emitRootExecutionJSONInputError(root, opts, "unknown_subcommand", err.Error())
		}
	}
	return err
}

// RemovedRootFlagSuggestion returns deterministic replacement guidance for
// historical root-level one-shot flags that are intentionally not registered on
// the Cobra tree anymore.
func RemovedRootFlagSuggestion(args []string) (string, bool) {
	for i, arg := range args {
		switch {
		case arg == "--oneshot":
			if i+1 < len(args) {
				return fmt.Sprintf("unknown flag: --oneshot; use `gormes chat -q %q`", args[i+1]), true
			}
			return "unknown flag: --oneshot; use `gormes chat -q \"your prompt\"`", true
		case strings.HasPrefix(arg, "--oneshot="):
			prompt := strings.TrimPrefix(arg, "--oneshot=")
			if prompt == "" {
				return "unknown flag: --oneshot; use `gormes chat -q \"your prompt\"`", true
			}
			return fmt.Sprintf("unknown flag: --oneshot; use `gormes chat -q %q`", prompt), true
		case arg == "-z":
			if i+1 < len(args) {
				return fmt.Sprintf("unknown shorthand flag: -z; use `gormes chat -q %q`", args[i+1]), true
			}
			return "unknown shorthand flag: -z; use `gormes chat -q \"your prompt\"`", true
		case strings.HasPrefix(arg, "-z") && len(arg) > 2:
			prompt := strings.TrimPrefix(arg, "-z")
			return fmt.Sprintf("unknown shorthand flag: -z; use `gormes chat -q %q`", prompt), true
		}
	}
	return "", false
}

func rootExecutionTypoSuggestion(opts RootExecutionOptions, args []string) (string, bool) {
	if opts.TypoSuggestion != nil {
		return opts.TypoSuggestion(args)
	}
	return cli.TypoSuggestion(args)
}

func emitRootExecutionJSONInputError(cmd *cobra.Command, opts RootExecutionOptions, action, errMsg string) error {
	err := EmitJSONInputError(cmd, action, errMsg, rootExecutionBuildProvenance(opts))
	return rootExecutionExitError(opts, 1, err)
}

func rootExecutionBuildProvenance(opts RootExecutionOptions) BuildProvenance {
	if opts.BuildProvenance == nil {
		return BuildProvenance{}
	}
	return opts.BuildProvenance()
}

func rootExecutionExitError(opts RootExecutionOptions, code int, err error) error {
	if opts.ExitCodeError != nil {
		return opts.ExitCodeError(code, err)
	}
	return NewExitCodeError(code, err)
}

func rootExecutionHandledExitError(opts RootExecutionOptions, err error) bool {
	if opts.HandledExitError != nil {
		return opts.HandledExitError(err)
	}
	var coded interface{ ExitCode() int }
	return errors.As(err, &coded)
}

// isCobraUnknownCommandError matches cobra's Find()/findSuggestions
// `unknown command "X" for "Y"[; did you mean "Z"?]` error message
// pattern. Cobra returns this as a plain `errors.New(...)` value with no
// wrapped sentinel — substring match is the most stable contract.
func isCobraUnknownCommandError(err error) bool {
	if err == nil {
		return false
	}
	return strings.HasPrefix(err.Error(), `unknown command "`) && strings.Contains(err.Error(), `" for "`)
}

// isCobraUnknownJSONFlagError matches cobra's flag parser rejection of --json
// by a parent that consumed the command path before subcommand routing.
func isCobraUnknownJSONFlagError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "unknown flag: --json") || strings.Contains(msg, `unknown flag "--json"`)
}

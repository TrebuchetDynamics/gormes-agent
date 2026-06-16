package gormescli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// ParentUnknownSubcommandGuardOptions configures the shared guard installed on
// parent-only commands so typos fail loudly instead of printing help or falling
// through to a parent RunE.
type ParentUnknownSubcommandGuardOptions struct {
	BuildProvenance func() BuildProvenance
	ExitCodeError   func(int, error) error
}

// InstallParentUnknownSubcommandGuards recursively installs a RunE on commands
// that only group subcommands. The guard preserves Cobra's suggestion UX while
// also honoring hidden parent --json discovery/error contracts.
func InstallParentUnknownSubcommandGuards(cmd *cobra.Command, opts ParentUnknownSubcommandGuardOptions) {
	if cmd == nil {
		return
	}
	for _, child := range cmd.Commands() {
		InstallParentUnknownSubcommandGuards(child, opts)
	}
	if !cmd.HasSubCommands() || cmd.Run != nil || cmd.RunE != nil {
		return
	}
	cmd.SilenceUsage = true
	cmd.Args = nil
	cmd.FParseErrWhitelist.UnknownFlags = true
	// Register --json as a hidden parent-only flag so the no-args fallback path
	// can detect "operator wants JSON" without reaching for os.Args (broken in
	// tests). Hidden so it doesn't pollute the parent's --help text. Subcommands
	// with their own --json flag are unaffected: cobra's flag parsing happens at
	// the matched leaf command, not the traversed parent.
	if cmd.Flags().Lookup("json") == nil {
		cmd.Flags().Bool("json", false, "")
		_ = cmd.Flags().MarkHidden("json")
	}
	// cobra.Command.SuggestionsFor compares against SuggestionsMinimumDistance
	// literally, but the field stays at 0 until cobra's own findSuggestions
	// lazy-inits it to 2. We don't route through findSuggestions (it's
	// package-private), so a typo like `gormes session lst` would otherwise only
	// match the suggestByPrefix branch — `lst` is NOT a prefix of `list`, so no
	// suggestion ever fires. Setting the field explicitly here keeps
	// edit-distance-1 typos within the suggestion window, matching the "Did you
	// mean: config" UX `gormes confg` already gets at the root level.
	if cmd.SuggestionsMinimumDistance <= 0 {
		cmd.SuggestionsMinimumDistance = 2
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			var msg string
			if suggestions := cmd.SuggestionsFor(args[0]); len(suggestions) > 0 {
				msg = fmt.Sprintf("unknown command %q for %q; did you mean %q?", args[0], cmd.CommandPath(), suggestions[0])
			} else {
				msg = fmt.Sprintf("unknown command %q for %q", args[0], cmd.CommandPath())
			}
			if ArgsIncludeJSONFlag(args) {
				return emitParentJSONInputError(cmd, opts, "unknown_subcommand", msg)
			}
			return fmt.Errorf("%s", msg)
		}
		// No subcommand provided. With --json the operator wants machine-readable
		// output, not Help text — emit a structured `subcommand_required` document
		// listing the available subcommands so fleet automation can discover the
		// parent's surface programmatically.
		if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
			return emitParentJSONSubcommandRequired(cmd, opts)
		}
		return cmd.Help()
	}
}

func emitParentJSONInputError(cmd *cobra.Command, opts ParentUnknownSubcommandGuardOptions, action, errMsg string) error {
	err := EmitJSONInputError(cmd, action, errMsg, parentGuardBuildProvenance(opts))
	return parentGuardExitError(opts, 1, err)
}

func emitParentJSONSubcommandRequired(cmd *cobra.Command, opts ParentUnknownSubcommandGuardOptions) error {
	available := make([]string, 0, len(cmd.Commands()))
	for _, child := range cmd.Commands() {
		if child.Hidden || child.Name() == "help" {
			continue
		}
		available = append(available, child.Name())
	}
	parent := cmd.CommandPath()
	report := struct {
		Build     BuildProvenance `json:"build"`
		Action    string          `json:"action"`
		Parent    string          `json:"parent"`
		Available []string        `json:"available"`
		Error     string          `json:"error"`
	}{
		Build:     parentGuardBuildProvenance(opts),
		Action:    "subcommand_required",
		Parent:    parent,
		Available: available,
		Error:     fmt.Sprintf("subcommand required for %q; choose one of: %s", parent, strings.Join(available, ", ")),
	}
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(report)
	return parentGuardExitError(opts, 1, fmt.Errorf("%s", report.Error))
}

func parentGuardBuildProvenance(opts ParentUnknownSubcommandGuardOptions) BuildProvenance {
	if opts.BuildProvenance == nil {
		return BuildProvenance{}
	}
	return opts.BuildProvenance()
}

func parentGuardExitError(opts ParentUnknownSubcommandGuardOptions, code int, err error) error {
	if opts.ExitCodeError != nil {
		return opts.ExitCodeError(code, err)
	}
	return NewExitCodeError(code, err)
}

package gormescli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// VisibleHelpCommandOptions configures the root help command installed by the
// executable shim. It keeps the CLI-platform help behavior importable without
// forcing gormescli to know cmd/gormes' concrete exit-code wrapper.
type VisibleHelpCommandOptions struct {
	ExitCodeError func(int, error) error
}

// InstallVisibleHelpCommand replaces Cobra's default hidden help command with a
// visible command that only resolves non-hidden command paths. Removed command
// names therefore fail as unknown help topics instead of leaking deprecated
// shortcuts through Cobra's default resolver.
func InstallVisibleHelpCommand(root *cobra.Command, opts VisibleHelpCommandOptions) {
	root.SetHelpCommand(&cobra.Command{
		Use:   "help [command]",
		Short: "Help about any command",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			target, ok := resolveVisibleHelpPath(root, args)
			if !ok {
				topic := strings.TrimSpace(strings.Join(args, " "))
				if topic == "" {
					topic = root.Name()
				}
				return visibleHelpExitError(opts, 2, fmt.Errorf("unknown help topic %q", topic))
			}
			target.SetOut(cmd.OutOrStdout())
			target.SetErr(cmd.ErrOrStderr())
			return target.Help()
		},
	})
}

func resolveVisibleHelpPath(root *cobra.Command, args []string) (*cobra.Command, bool) {
	if root == nil {
		return nil, false
	}
	current := root
	for _, arg := range args {
		part := strings.TrimSpace(arg)
		if part == "" {
			continue
		}
		var next *cobra.Command
		for _, child := range current.Commands() {
			if child.Hidden || child.Name() == "help" {
				continue
			}
			if child.Name() == part || visibleHelpCommandHasAlias(child, part) {
				next = child
				break
			}
		}
		if next == nil {
			return nil, false
		}
		current = next
	}
	return current, true
}

func visibleHelpCommandHasAlias(cmd *cobra.Command, alias string) bool {
	for _, candidate := range cmd.Aliases {
		if candidate == alias {
			return true
		}
	}
	return false
}

// InstallRootHelpRenderer renders the root Long text before Cobra's generated
// command/flag usage. This keeps Gormes' first-run operator guide visible while
// retaining Cobra's live command inventory.
func InstallRootHelpRenderer(root *cobra.Command) {
	root.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		usage := strings.TrimRightFunc(firstHelpText(cmd.Long, cmd.Short), func(r rune) bool {
			return r == ' ' || r == '\t' || r == '\r' || r == '\n'
		})
		if usage != "" {
			fmt.Fprintln(cmd.OutOrStdout(), usage)
			fmt.Fprintln(cmd.OutOrStdout())
		}
		if cmd.Runnable() || cmd.HasSubCommands() {
			fmt.Fprint(cmd.OutOrStdout(), cmd.UsageString())
		}
	})
}

func firstHelpText(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func visibleHelpExitError(opts VisibleHelpCommandOptions, code int, err error) error {
	if opts.ExitCodeError != nil {
		return opts.ExitCodeError(code, err)
	}
	return NewExitCodeError(code, err)
}

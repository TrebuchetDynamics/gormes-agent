package gormescli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	appsetup "github.com/TrebuchetDynamics/gormes-agent/internal/app/setup"
	appsetupchoice "github.com/TrebuchetDynamics/gormes-agent/internal/app/setupchoice"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
	tuilocal "github.com/TrebuchetDynamics/gormes-agent/internal/tui/local"
)

type ToolsCommandOptions struct {
	ConfigPath func() string
}

func (o ToolsCommandOptions) configPath() string {
	if o.ConfigPath != nil {
		return o.ConfigPath()
	}
	return config.ConfigPath()
}

func NewToolsCommand(opts ToolsCommandOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "tools",
		Short:        "Manage Hermes-compatible tool allowlists",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runToolsListCommand(cmd, opts)
		},
	}
	cmd.AddCommand(&cobra.Command{
		Use:          "list",
		Short:        "List tool availability",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runToolsListCommand(cmd, opts)
		},
	})
	cmd.AddCommand(newToolsConfigureCommand("enable", "Enable a tool"))
	cmd.AddCommand(newToolsConfigureCommand("disable", "Disable a tool"))
	return cmd
}

func newToolsConfigureCommand(action, short string) *cobra.Command {
	return &cobra.Command{
		Use:          action + " <name> [name...]",
		Short:        short,
		Args:         cobra.MinimumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runToolsConfigureCommand(cmd, action, args)
		},
	}
}

func runToolsListCommand(cmd *cobra.Command, opts ToolsCommandOptions) error {
	_, toolCfg, err := appsetup.LoadToolsConfig(opts.configPath())
	if err != nil {
		return err
	}
	status, err := toolCfg.PlatformStatus("cli")
	if err != nil {
		return err
	}
	enabled := toolCommandStringSet(status.RuntimeToolsets)
	options, err := appsetup.ToolOptions()
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Tools for CLI")
	for _, option := range options {
		state := "disabled"
		if enabled[appsetupchoice.NormalizeValue(option.Key)] {
			state = "enabled"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%-18s %s  %s\n", option.Key, state, option.Label)
	}
	return nil
}

func runToolsConfigureCommand(cmd *cobra.Command, action string, names []string) error {
	result, err := tuilocal.ConfigureTools(tui.ToolsConfigureRequest{Action: action, Names: names, SessionID: "cli-tools-command"})
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	label := action + "d"
	if action == "disable" {
		label = "disabled"
	}
	if len(result.Changed) > 0 {
		fmt.Fprintf(out, "%s: %s\n", label, strings.Join(result.Changed, ", "))
	} else {
		fmt.Fprintf(out, "%s: none\n", label)
	}
	if len(result.Unknown) > 0 {
		fmt.Fprintf(out, "unknown: %s\n", strings.Join(result.Unknown, ", "))
	}
	if len(result.MissingServers) > 0 {
		fmt.Fprintf(out, "missing_mcp_servers: %s\n", strings.Join(result.MissingServers, ", "))
	}
	fmt.Fprintf(out, "session_reset_required=%t\n", result.Reset)
	return nil
}

func toolCommandStringSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}

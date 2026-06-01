package main

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
	"github.com/spf13/cobra"
)

const (
	hermesGatewayCronRow = "Gateway, platform, webhook, and cron management CLI"
	hermesDiagnosticsRow = "Diagnostics, backup, logs, and status CLI"
	hermesConfigRow      = "Hermes config migration dry-run manifest"
	hermesToolRow        = "Tool/runtime/security rows"
	hermesACPMCPRow      = "ACP server side"
	hermesSkillsRow      = "Skills hub direct URL install name/category guard"
	hermesMemoryRow      = "Goncho memory integration into normal agent turn"
	hermesKanbanRow      = "Hermes Kanban durable board core"
)

func newDumpCommand() *cobra.Command {
	return newHermesUnavailableCommand(hermesUnavailableCommandSpec{
		Use:   "dump",
		Short: "Collect a Hermes-compatible debug dump",
		Row:   hermesDiagnosticsRow,
	})
}

func newDebugCommand() *cobra.Command {
	return newHermesUnavailableParent(
		"debug",
		"Manage Hermes-compatible debug share bundles",
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "share",
			Short: "Share a debug bundle",
			Row:   hermesDiagnosticsRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:         "delete",
			Short:       "Delete a shared debug bundle",
			Row:         hermesDiagnosticsRow,
			Destructive: true,
		}),
	)
}

func newBackupCommand() *cobra.Command {
	return newHermesUnavailableCommand(hermesUnavailableCommandSpec{
		Use:   "backup",
		Short: "Create a Hermes-compatible backup archive",
		Row:   "Backup/update opt-in and exclusion policy",
	})
}

func newImportCommand() *cobra.Command {
	return newHermesUnavailableCommand(hermesUnavailableCommandSpec{
		Use:   "import",
		Short: "Import a Hermes configuration or state archive",
		Row:   hermesConfigRow,
	})
}

func newToolsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "tools",
		Short:        "Manage Hermes-compatible tool allowlists",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runToolsListCommand(cmd)
		},
	}
	cmd.AddCommand(&cobra.Command{
		Use:          "list",
		Short:        "List tool availability",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runToolsListCommand(cmd)
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

func runToolsListCommand(cmd *cobra.Command) error {
	_, toolCfg, err := loadSetupToolsConfig(config.ConfigPath())
	if err != nil {
		return err
	}
	status, err := toolCfg.PlatformStatus("cli")
	if err != nil {
		return err
	}
	enabled := stringSet(status.RuntimeToolsets)
	options, err := setupToolOptions()
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Tools for CLI")
	for _, option := range options {
		state := "disabled"
		if enabled[normalizeSetupChoice(option.key)] {
			state = "enabled"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%-18s %s  %s\n", option.key, state, option.label)
	}
	return nil
}

func runToolsConfigureCommand(cmd *cobra.Command, action string, names []string) error {
	result, err := configureTUITools(tui.ToolsConfigureRequest{Action: action, Names: names, SessionID: "cli-tools-command"})
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

package main

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	"github.com/spf13/cobra"
)

const (
	hermesGatewayCronRow = "Gateway, platform, webhook, and cron management CLI"
	hermesDiagnosticsRow = "Diagnostics, backup, logs, and status CLI"
	hermesConfigRow      = "Hermes config migration dry-run manifest"
	hermesToolRow        = "Tool/runtime/security rows"
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
	return gormescli.NewToolsCommand(gormescli.ToolsCommandOptions{})
}

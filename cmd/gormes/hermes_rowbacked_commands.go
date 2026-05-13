package main

import "github.com/spf13/cobra"

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

func newPairingCommand() *cobra.Command {
	return newHermesUnavailableParent(
		"pairing",
		"Manage gateway pairing requests",
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "list",
			Short: "List pending pairing requests",
			Row:   hermesGatewayCronRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "approve <id>",
			Short: "Approve a pairing request",
			Row:   hermesGatewayCronRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:         "revoke <id>",
			Short:       "Revoke an approved pairing",
			Row:         hermesGatewayCronRow,
			Destructive: true,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:         "clear-pending",
			Short:       "Clear pending pairing requests",
			Row:         hermesGatewayCronRow,
			Destructive: true,
			FlagSet:     hermesUnavailableYesFlag,
		}),
	)
}

func newWebhookCommand() *cobra.Command {
	return newHermesUnavailableParent(
		"webhook",
		"Manage Hermes-compatible webhook subscriptions",
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:     "subscribe <target>",
			Aliases: []string{"add"},
			Short:   "Subscribe a webhook target",
			Row:     hermesGatewayCronRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:     "list",
			Aliases: []string{"ls"},
			Short:   "List webhook subscriptions",
			Row:     hermesGatewayCronRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:         "remove <id>",
			Aliases:     []string{"rm"},
			Short:       "Remove a webhook subscription",
			Row:         hermesGatewayCronRow,
			Destructive: true,
			FlagSet:     hermesUnavailableYesFlag,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "test <id>",
			Short: "Send a test webhook event",
			Row:   hermesGatewayCronRow,
		}),
	)
}

func newHooksCommand() *cobra.Command {
	return newHermesUnavailableParent(
		"hooks",
		"Manage local Hermes-compatible hook registrations",
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:     "list",
			Aliases: []string{"ls"},
			Short:   "List hook registrations",
			Row:     hermesGatewayCronRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "test <name>",
			Short: "Test a hook registration",
			Row:   hermesGatewayCronRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:         "revoke <name>",
			Aliases:     []string{"remove", "rm"},
			Short:       "Revoke a hook registration",
			Row:         hermesGatewayCronRow,
			Destructive: true,
			FlagSet:     hermesUnavailableYesFlag,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "doctor",
			Short: "Inspect hook configuration health",
			Row:   hermesGatewayCronRow,
		}),
	)
}

func newToolsCommand() *cobra.Command {
	return newHermesUnavailableParent(
		"tools",
		"Manage Hermes-compatible tool allowlists",
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "list",
			Short: "List tool availability",
			Row:   hermesToolRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "disable <name>",
			Short: "Disable a tool",
			Row:   hermesToolRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "enable <name>",
			Short: "Enable a tool",
			Row:   hermesToolRow,
		}),
	)
}

func newInsightsCommand() *cobra.Command {
	return newHermesUnavailableCommand(hermesUnavailableCommandSpec{
		Use:   "insights",
		Short: "Show Hermes-compatible runtime insights",
		Row:   "Self-monitoring telemetry",
	})
}

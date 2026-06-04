package main

import (
	"github.com/spf13/cobra"

	cronapp "github.com/TrebuchetDynamics/gormes-agent/internal/app/cron"
)

func newCronCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cron",
		Short: "Manage scheduled cron jobs",
		Long: `List, inspect, and remove cron jobs managed by the native cron scheduler.

Cron jobs run agent prompts or shell scripts on a schedule and deliver
results to configured targets (Telegram, Slack, etc.).

Examples:
  gormes cron list              # show all cron jobs
  gormes cron status <job-id>   # show job details
  gormes cron remove <job-id>   # remove a cron job
`,
	}

	dbFlag := cmd.PersistentFlags().String("db", "",
		"Path to session DB (default: autodetect from GORMES_HOME)")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all cron jobs with status and last run time",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, smap, err := cronapp.OpenStore(*dbFlag)
			if err != nil {
				return err
			}
			defer smap.Close()
			return cronapp.ListJobs(cmd.OutOrStdout(), store)
		},
	}

	removeCmd := &cobra.Command{
		Use:     "remove <job-id>",
		Aliases: []string{"rm", "delete"},
		Short:   "Remove a cron job by ID (prefix matching supported)",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, smap, err := cronapp.OpenStore(*dbFlag)
			if err != nil {
				return err
			}
			defer smap.Close()
			return cronapp.RemoveJob(cmd.OutOrStdout(), store, args[0])
		},
	}

	statusCmd := &cobra.Command{
		Use:   "status <job-id>",
		Short: "Show detailed job info and recent run history",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, runStore, smap, err := cronapp.OpenStoreWithRuns(*dbFlag)
			if err != nil {
				return err
			}
			if smap != nil {
				defer smap.Close()
			}
			return cronapp.StatusJob(cmd.OutOrStdout(), store, runStore, args[0])
		},
	}

	cmd.AddCommand(
		listCmd,
		removeCmd,
		statusCmd,
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:     "create",
			Aliases: []string{"add"},
			Short:   "Create a scheduled cron job",
			Row:     hermesGatewayCronRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "edit <job-id>",
			Short: "Edit a scheduled cron job",
			Row:   hermesGatewayCronRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "pause <job-id>",
			Short: "Pause a scheduled cron job",
			Row:   hermesGatewayCronRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "resume <job-id>",
			Short: "Resume a scheduled cron job",
			Row:   hermesGatewayCronRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "run <job-id>",
			Short: "Run a scheduled cron job now",
			Row:   hermesGatewayCronRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "tick",
			Short: "Run one scheduler tick",
			Row:   hermesGatewayCronRow,
		}),
	)
	return cmd
}

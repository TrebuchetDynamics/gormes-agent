package gormescli

import (
	"github.com/spf13/cobra"

	cronapp "github.com/TrebuchetDynamics/gormes-agent/internal/app/cron"
)

type CronCommandRows struct {
	Create RowBackedCommandSpec
	Edit   RowBackedCommandSpec
	Pause  RowBackedCommandSpec
	Resume RowBackedCommandSpec
	Run    RowBackedCommandSpec
	Tick   RowBackedCommandSpec
}

type CronCommandOptions struct {
	Rows               CronCommandRows
	UnavailableCommand func(RowBackedCommandSpec) *cobra.Command
}

func NewCronCommand(opts CronCommandOptions) *cobra.Command {
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

	rows := cronCommandRowsWithDefaults(opts.Rows)
	cmd.AddCommand(
		listCmd,
		removeCmd,
		statusCmd,
		cronUnavailableCommand(opts, rows.Create),
		cronUnavailableCommand(opts, rows.Edit),
		cronUnavailableCommand(opts, rows.Pause),
		cronUnavailableCommand(opts, rows.Resume),
		cronUnavailableCommand(opts, rows.Run),
		cronUnavailableCommand(opts, rows.Tick),
	)
	return cmd
}

func cronCommandRowsWithDefaults(rows CronCommandRows) CronCommandRows {
	if rows.Create.Use == "" {
		rows.Create.Use = "create"
		rows.Create.Aliases = []string{"add"}
		rows.Create.Short = "Create a scheduled cron job"
	}
	if rows.Edit.Use == "" {
		rows.Edit.Use = "edit <job-id>"
		rows.Edit.Short = "Edit a scheduled cron job"
	}
	if rows.Pause.Use == "" {
		rows.Pause.Use = "pause <job-id>"
		rows.Pause.Short = "Pause a scheduled cron job"
	}
	if rows.Resume.Use == "" {
		rows.Resume.Use = "resume <job-id>"
		rows.Resume.Short = "Resume a scheduled cron job"
	}
	if rows.Run.Use == "" {
		rows.Run.Use = "run <job-id>"
		rows.Run.Short = "Run a scheduled cron job now"
	}
	if rows.Tick.Use == "" {
		rows.Tick.Use = "tick"
		rows.Tick.Short = "Run one scheduler tick"
	}
	return rows
}

func cronUnavailableCommand(opts CronCommandOptions, spec RowBackedCommandSpec) *cobra.Command {
	if opts.UnavailableCommand != nil {
		return opts.UnavailableCommand(spec)
	}
	return NewRowBackedCommand(spec, RowBackedCommandOptions{})
}

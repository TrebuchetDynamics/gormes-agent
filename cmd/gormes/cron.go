package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/automation/cron"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
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
			store, smap, err := openCronStore(*dbFlag)
			if err != nil {
				return err
			}
			defer smap.Close()
			return runCronList(cmd.OutOrStdout(), store)
		},
	}

	removeCmd := &cobra.Command{
		Use:     "remove <job-id>",
		Aliases: []string{"rm", "delete"},
		Short:   "Remove a cron job by ID (prefix matching supported)",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, smap, err := openCronStore(*dbFlag)
			if err != nil {
				return err
			}
			defer smap.Close()
			return runCronRemove(cmd.OutOrStdout(), store, args[0])
		},
	}

	statusCmd := &cobra.Command{
		Use:   "status <job-id>",
		Short: "Show detailed job info and recent run history",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, runStore, smap, err := openCronStoreWithRuns(*dbFlag)
			if err != nil {
				return err
			}
			if smap != nil {
				defer smap.Close()
			}
			return runCronStatus(cmd.OutOrStdout(), store, runStore, args[0])
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

func openCronStore(dbPath string) (*cron.Store, *session.BoltMap, error) {
	if dbPath == "" {
		dbPath = config.SessionDBPath()
	}
	smap, err := session.OpenBolt(dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open session DB %s: %w", dbPath, err)
	}
	store, err := cron.NewStore(smap.DB())
	if err != nil {
		smap.Close()
		return nil, nil, fmt.Errorf("open cron store: %w", err)
	}
	return store, smap, nil
}

func openCronStoreWithRuns(dbPath string) (*cron.Store, *cron.RunStore, *session.BoltMap, error) {
	if dbPath == "" {
		dbPath = config.SessionDBPath()
	}
	smap, err := session.OpenBolt(dbPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open session DB %s: %w", dbPath, err)
	}
	store, err := cron.NewStore(smap.DB())
	if err != nil {
		smap.Close()
		return nil, nil, nil, fmt.Errorf("open cron store: %w", err)
	}
	return store, nil, smap, nil // RunStore needs sql.DB; not wired here yet
}

func runCronList(out io.Writer, store *cron.Store) error {
	return gormescli.RunCronList(out, store)
}

func runCronRemove(out io.Writer, store *cron.Store, jobID string) error {
	return gormescli.RunCronRemove(out, store, jobID)
}

func runCronStatus(out io.Writer, store *cron.Store, runStore *cron.RunStore, jobID string) error {
	return gormescli.RunCronStatus(out, store, runStore, jobID)
}

func findJob(store *cron.Store, id string) *cron.Job {
	return gormescli.FindCronJob(store, id)
}

package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/cron"
	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
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
			return runCronList(store)
		},
	}

	removeCmd := &cobra.Command{
		Use:   "remove <job-id>",
		Short: "Remove a cron job by ID (prefix matching supported)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, smap, err := openCronStore(*dbFlag)
			if err != nil {
				return err
			}
			defer smap.Close()
			return runCronRemove(store, args[0])
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
			return runCronStatus(store, runStore, args[0])
		},
	}

	cmd.AddCommand(listCmd, removeCmd, statusCmd)
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

func runCronList(store *cron.Store) error {
	jobs, err := store.List()
	if err != nil {
		return fmt.Errorf("list cron jobs: %w", err)
	}
	if len(jobs) == 0 {
		fmt.Println("No cron jobs found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tName\tSchedule\tStatus\tLast Run\tScript/Prompt\tPaused")
	fmt.Fprintln(w, "--\t----\t--------\t------\t--------\t------------\t------")
	for _, j := range jobs {
		lastRun := "never"
		if j.LastRunUnix > 0 {
			lastRun = time.Unix(j.LastRunUnix, 0).Format("2006-01-02 15:04")
		}
		status := j.LastStatus
		if status == "" {
			status = "-"
		}
		paused := ""
		if j.Paused {
			paused = "yes"
		}
		desc := j.Script
		if desc == "" {
			desc = strings.Split(j.Prompt, "\n")[0]
			if len(desc) > 50 {
				desc = desc[:47] + "..."
			}
		} else {
			if len(desc) > 50 {
				desc = "..." + desc[len(desc)-47:]
			}
		}
		if j.NoAgent {
			desc += " [script]"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			j.ID[:8], j.Name, j.Schedule, status, lastRun, desc, paused)
	}
	return w.Flush()
}

func runCronRemove(store *cron.Store, jobID string) error {
	job := findJob(store, jobID)
	if job == nil {
		return fmt.Errorf("cron job %q not found", jobID)
	}
	if err := store.Delete(job.ID); err != nil {
		return fmt.Errorf("delete cron job %s: %w", job.ID[:8], err)
	}
	fmt.Printf("Removed cron job %s (%s)\n", job.ID[:8], job.Name)
	return nil
}

func runCronStatus(store *cron.Store, runStore *cron.RunStore, jobID string) error {
	job := findJob(store, jobID)
	if job == nil {
		return fmt.Errorf("cron job %q not found", jobID)
	}

	fmt.Printf("Job:      %s (%s)\n", job.Name, job.ID[:8])
	fmt.Printf("Schedule: %s\n", job.Schedule)
	fmt.Printf("Created:  %s\n", time.Unix(job.CreatedAt, 0).Format("2006-01-02 15:04:05"))
	fmt.Printf("Paused:   %v\n", job.Paused)
	if job.Script != "" {
		fmt.Printf("Script:   %s\n", job.Script)
	}
	fmt.Printf("NoAgent:  %v\n", job.NoAgent)
	if job.Deliver != "" {
		fmt.Printf("Deliver:  %s\n", job.Deliver)
	}
	if job.LastRunUnix > 0 {
		fmt.Printf("Last run: %s (status: %s)\n",
			time.Unix(job.LastRunUnix, 0).Format("2006-01-02 15:04:05"), job.LastStatus)
	} else {
		fmt.Println("Last run: never")
	}

	// Show last 5 runs from run store
	if runStore != nil {
		runs, err := runStore.LatestRuns(nil, job.ID, 5)
		if err == nil && len(runs) > 0 {
			fmt.Println("\nRecent runs:")
			for _, r := range runs {
				started := time.Unix(r.StartedAt, 0).Format("15:04:05")
				preview := r.OutputPreview
				if len(preview) > 80 {
					preview = preview[:77] + "..."
				}
				errFlag := ""
				if r.ErrorMsg != "" {
					errFlag = " ERROR"
				}
				fmt.Printf("  %s  %-12s %s%s\n", started, r.Status, preview, errFlag)
			}
		}
	}
	return nil
}

func findJob(store *cron.Store, id string) *cron.Job {
	jobs, err := store.List()
	if err != nil {
		return nil
	}
	for _, j := range jobs {
		if j.ID == id || j.ID[:8] == id {
			return &j
		}
	}
	return nil
}


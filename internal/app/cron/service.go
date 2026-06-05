package cron

import (
	"context"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	autocron "github.com/TrebuchetDynamics/gormes-agent/internal/automation/cron"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
)

type JobStore interface {
	List() ([]autocron.Job, error)
	Delete(id string) error
}

type RunReader interface {
	LatestRuns(ctx context.Context, jobID string, limit int) ([]autocron.Run, error)
}

func OpenStore(dbPath string) (*autocron.Store, *session.BoltMap, error) {
	if dbPath == "" {
		dbPath = config.SessionDBPath()
	}
	smap, err := session.OpenBolt(dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open session DB %s: %w", dbPath, err)
	}
	store, err := autocron.NewStore(smap.DB())
	if err != nil {
		smap.Close()
		return nil, nil, fmt.Errorf("open cron store: %w", err)
	}
	return store, smap, nil
}

func OpenStoreWithRuns(dbPath string) (*autocron.Store, *autocron.RunStore, *session.BoltMap, error) {
	if dbPath == "" {
		dbPath = config.SessionDBPath()
	}
	smap, err := session.OpenBolt(dbPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open session DB %s: %w", dbPath, err)
	}
	store, err := autocron.NewStore(smap.DB())
	if err != nil {
		smap.Close()
		return nil, nil, nil, fmt.Errorf("open cron store: %w", err)
	}
	return store, nil, smap, nil // RunStore needs sql.DB; not wired here yet
}

func ListJobs(out io.Writer, store JobStore) error {
	jobs, err := store.List()
	if err != nil {
		return fmt.Errorf("list cron jobs: %w", err)
	}
	if len(jobs) == 0 {
		fmt.Fprintln(out, "No cron jobs found.")
		return nil
	}

	w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
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

func RemoveJob(out io.Writer, store JobStore, jobID string) error {
	job := FindJob(store, jobID)
	if job == nil {
		return fmt.Errorf("cron job %q not found", jobID)
	}
	if err := store.Delete(job.ID); err != nil {
		return fmt.Errorf("delete cron job %s: %w", job.ID[:8], err)
	}
	fmt.Fprintf(out, "Removed cron job %s (%s)\n", job.ID[:8], job.Name)
	return nil
}

func StatusJob(out io.Writer, store JobStore, runStore RunReader, jobID string) error {
	job := FindJob(store, jobID)
	if job == nil {
		return fmt.Errorf("cron job %q not found", jobID)
	}

	fmt.Fprintf(out, "Job:      %s (%s)\n", job.Name, job.ID[:8])
	fmt.Fprintf(out, "Schedule: %s\n", job.Schedule)
	fmt.Fprintf(out, "Created:  %s\n", time.Unix(job.CreatedAt, 0).Format("2006-01-02 15:04:05"))
	fmt.Fprintf(out, "Paused:   %v\n", job.Paused)
	if job.Script != "" {
		fmt.Fprintf(out, "Script:   %s\n", job.Script)
	}
	fmt.Fprintf(out, "NoAgent:  %v\n", job.NoAgent)
	if job.Deliver != "" {
		fmt.Fprintf(out, "Deliver:  %s\n", job.Deliver)
	}
	if job.LastRunUnix > 0 {
		fmt.Fprintf(out, "Last run: %s (status: %s)\n",
			time.Unix(job.LastRunUnix, 0).Format("2006-01-02 15:04:05"), job.LastStatus)
	} else {
		fmt.Fprintln(out, "Last run: never")
	}

	if runStore != nil {
		runs, err := runStore.LatestRuns(nil, job.ID, 5)
		if err == nil && len(runs) > 0 {
			fmt.Fprintln(out, "\nRecent runs:")
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
				fmt.Fprintf(out, "  %s  %-12s %s%s\n", started, r.Status, preview, errFlag)
			}
		}
	}
	return nil
}

func FindJob(store JobStore, id string) *autocron.Job {
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

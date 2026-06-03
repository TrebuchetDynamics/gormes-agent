package gormescli

import (
	"io"

	appcron "github.com/TrebuchetDynamics/gormes-agent/internal/app/cron"
	autocron "github.com/TrebuchetDynamics/gormes-agent/internal/automation/cron"
)

func RunCronList(out io.Writer, store appcron.JobStore) error {
	return appcron.ListJobs(out, store)
}

func RunCronRemove(out io.Writer, store appcron.JobStore, jobID string) error {
	return appcron.RemoveJob(out, store, jobID)
}

func RunCronStatus(out io.Writer, store appcron.JobStore, runStore appcron.RunReader, jobID string) error {
	return appcron.StatusJob(out, store, runStore, jobID)
}

func FindCronJob(store appcron.JobStore, id string) *autocron.Job {
	return appcron.FindJob(store, id)
}

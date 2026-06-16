package cron

import (
	"context"
	"database/sql"

	cronruns "github.com/TrebuchetDynamics/gormes-agent/internal/automation/cron/runs"
)

// Run is one scheduled fire's audit record. Written by the Executor after each
// run (success, timeout, error, or suppressed).
type Run = cronruns.Run

// RunStore writes to the cron_runs SQLite table. Read path is rare (CRON.md
// mirror only); writes happen once per job fire.
type RunStore struct {
	inner *cronruns.RunStore
	db    *sql.DB
}

// NewRunStore wraps an open *sql.DB. The cron_runs table must exist — it's
// created by migration 3d->3e (internal/memory/schema.go).
func NewRunStore(db *sql.DB) *RunStore {
	return &RunStore{inner: cronruns.NewRunStore(db), db: db}
}

// RecordRun persists one run.
func (s *RunStore) RecordRun(ctx context.Context, r Run) error {
	return s.inner.RecordRun(ctx, r)
}

// RecordRunWithID persists one run and returns the SQLite row id assigned to it.
func (s *RunStore) RecordRunWithID(ctx context.Context, r Run) (int64, error) {
	return s.inner.RecordRunWithID(ctx, r)
}

// LatestRuns returns up to limit most-recent runs for the given job_id.
func (s *RunStore) LatestRuns(ctx context.Context, jobID string, limit int) ([]Run, error) {
	return s.inner.LatestRuns(ctx, jobID, limit)
}

// LatestCompletedOutput returns the latest successful delivered output for a job.
func (s *RunStore) LatestCompletedOutput(ctx context.Context, jobID string) (string, bool, error) {
	return s.inner.LatestCompletedOutput(ctx, jobID)
}

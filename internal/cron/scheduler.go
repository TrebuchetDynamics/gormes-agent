package cron

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	rc "github.com/robfig/cron/v3"
)

// SchedulerConfig is the set of live dependencies.
type SchedulerConfig struct {
	Store    *Store // bbolt job persistence
	Executor Runner // interface — real *Executor or a test fake
}

// Scheduler owns a robfig *cron.Cron instance and the mapping of
// job IDs to registered EntryIDs. MVP is load-once at Start time;
// live reload on store mutations is a 2.D.2 concern.
type Scheduler struct {
	cfg     SchedulerConfig
	cron    *rc.Cron
	log     *slog.Logger
	mu      sync.Mutex
	entries map[string]rc.EntryID // jobID -> EntryID (for future Remove)
}

// NewScheduler constructs a Scheduler. Call Start to actually begin
// ticking. log may be nil (slog.Default used).
func NewScheduler(cfg SchedulerConfig, log *slog.Logger) *Scheduler {
	if log == nil {
		log = slog.Default()
	}
	return &Scheduler{
		cfg:     cfg,
		cron:    rc.New(rc.WithParser(rc.NewParser(rc.Minute | rc.Hour | rc.Dom | rc.Month | rc.Dow | rc.Descriptor))),
		log:     log,
		entries: make(map[string]rc.EntryID),
	}
}

// Start loads all non-paused jobs from the store, registers their cron
// expressions, and starts the ticker. Jobs with invalid schedules are
// skipped with a warning; other jobs continue as normal.
//
// Non-blocking: the cron ticker runs on its own goroutine. Stop must be
// called to tear down.
func (s *Scheduler) Start(ctx context.Context) error {
	jobs, err := s.cfg.Store.List()
	if err != nil {
		return fmt.Errorf("scheduler: list jobs: %w", err)
	}
	for _, job := range jobs {
		if job.Paused {
			continue
		}
		if vErr := ValidateSchedule(job.Schedule); vErr != nil {
			s.log.Warn("cron: skipping job with invalid schedule",
				"job_id", job.ID, "name", job.Name,
				"schedule", job.Schedule, "err", vErr)
			continue
		}
		jobCopy := job // capture for closure
		id, aErr := s.cron.AddFunc(job.Schedule, func() {
			defer func() {
				if r := recover(); r != nil {
					s.log.Warn("cron: panic in job",
						"job_id", jobCopy.ID, "name", jobCopy.Name, "panic", r)
				}
			}()
			s.cfg.Executor.Run(ctx, jobCopy)
		})
		if aErr != nil {
			s.log.Warn("cron: AddFunc failed",
				"job_id", job.ID, "name", job.Name, "err", aErr)
			continue
		}
		s.mu.Lock()
		s.entries[job.ID] = id
		s.mu.Unlock()
	}
	s.cron.Start()
	return nil
}

// Stop halts the ticker and waits for any running jobs (bounded by ctx).
// Idempotent — safe to call before or after Start.
func (s *Scheduler) Stop(ctx context.Context) {
	if s.cron == nil {
		return
	}
	stopped := s.cron.Stop() // returns a context that's Done when running jobs finish
	select {
	case <-stopped.Done():
	case <-ctx.Done():
	}
}

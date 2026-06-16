package cron

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	cronlock "github.com/TrebuchetDynamics/gormes-agent/internal/automation/cron/lock"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
	rc "github.com/robfig/cron/v3"
)

// SchedulerConfig is the set of live dependencies.
type SchedulerConfig struct {
	Store            *Store // bbolt job persistence
	Executor         Runner // interface — real *Executor or a test fake
	MCPOrphanCleanup func()
	// LockPath points at the cross-process scheduler tick lock. Empty disables
	// locking for direct unit-test runTick calls; Start fills the Hermes-compatible
	// default <GORMES_HOME>/cron/.tick.lock.
	LockPath string
}

// Scheduler owns a robfig *cron.Cron instance and the mapping of
// job IDs to registered EntryIDs. MVP is load-once at Start time;
// live reload on store mutations is a 2.D.2 concern.
type Scheduler struct {
	cfg     SchedulerConfig
	cron    *rc.Cron
	log     *slog.Logger
	mu      sync.Mutex
	tickMu  sync.Mutex
	entries map[string]rc.EntryID // jobID -> EntryID (for future Remove)
}

// NewScheduler constructs a Scheduler. Call Start to actually begin
// ticking. log may be nil (slog.Default used).
func NewScheduler(cfg SchedulerConfig, log *slog.Logger) *Scheduler {
	if log == nil {
		log = slog.Default()
	}
	if cfg.MCPOrphanCleanup == nil {
		cfg.MCPOrphanCleanup = func() {
			tools.ReapMCPStdioOrphans()
		}
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
	if strings.TrimSpace(s.cfg.LockPath) == "" {
		s.cfg.LockPath = cronlock.DefaultPath()
	}
	jobs, err := s.cfg.Store.List()
	if err != nil {
		return fmt.Errorf("scheduler: list jobs: %w", err)
	}
	jobsBySchedule := make(map[string][]Job)
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
		jobsBySchedule[job.Schedule] = append(jobsBySchedule[job.Schedule], job)
	}
	for schedule, scheduleJobs := range jobsBySchedule {
		tickJobs := append([]Job(nil), scheduleJobs...)
		id, aErr := s.cron.AddFunc(schedule, func() {
			s.runTick(ctx, tickJobs)
		})
		if aErr != nil {
			for _, job := range tickJobs {
				s.log.Warn("cron: AddFunc failed",
					"job_id", job.ID, "name", job.Name, "err", aErr)
			}
			continue
		}
		s.mu.Lock()
		for _, job := range tickJobs {
			s.entries[job.ID] = id
		}
		s.mu.Unlock()
	}
	s.cron.Start()
	return nil
}

func (s *Scheduler) runTick(ctx context.Context, jobs []Job) {
	if strings.TrimSpace(s.cfg.LockPath) != "" {
		s.tickMu.Lock()
		defer s.tickMu.Unlock()

		lock, acquired, err := cronlock.Acquire(s.cfg.LockPath)
		if err != nil {
			s.log.Debug("cron: tick skipped; lock unavailable", "lock_path", s.cfg.LockPath, "err", err)
			return
		}
		if !acquired {
			s.log.Debug("cron: tick skipped; another scheduler holds the lock", "lock_path", s.cfg.LockPath)
			return
		}
		defer lock.Release()
	}

	var parallel []Job
	for _, job := range jobs {
		jobCopy := job
		if strings.TrimSpace(job.Workdir) != "" {
			s.runOneJob(ctx, jobCopy)
			continue
		}
		parallel = append(parallel, jobCopy)
	}
	var wg sync.WaitGroup
	for _, job := range parallel {
		jobCopy := job
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.runOneJob(ctx, jobCopy)
		}()
	}
	wg.Wait()
	s.cfg.MCPOrphanCleanup()
}

func (s *Scheduler) runOneJob(ctx context.Context, job Job) {
	defer func() {
		if r := recover(); r != nil {
			s.log.Warn("cron: panic in job",
				"job_id", job.ID, "name", job.Name, "panic", r)
		}
	}()
	s.cfg.Executor.Run(ctx, job)
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

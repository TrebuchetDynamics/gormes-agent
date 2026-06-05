package cron

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestCronRunStateParallelWritesSerialize(t *testing.T) {
	e, _, cleanup := newTestExecutorEnv(t, newFakeKernel("unused", 0))
	defer cleanup()

	job := NewJob("parallel-state", "every 1h", "p")
	job.Repeat = 10
	if err := e.cfg.JobStore.Create(job); err != nil {
		t.Fatalf("Create job: %v", err)
	}

	start := time.Date(2026, 5, 6, 8, 0, 0, 0, time.UTC).Unix()
	runs := []Run{
		{JobID: job.ID, StartedAt: start + 1, FinishedAt: start + 11, PromptHash: "h1", Status: "success", OutputPreview: "first"},
		{JobID: job.ID, StartedAt: start + 3, FinishedAt: start + 13, PromptHash: "h3", Status: "suppressed", SuppressionReason: "silent"},
		{JobID: job.ID, StartedAt: start + 2, FinishedAt: start + 12, PromptHash: "h2", Status: "error", ErrorMsg: "boom"},
	}

	var wg sync.WaitGroup
	for _, run := range runs {
		run := run
		wg.Add(1)
		go func() {
			defer wg.Done()
			e.recordAndUpdateJob(context.Background(), job, run)
		}()
	}
	wg.Wait()

	got, err := e.cfg.JobStore.Get(job.ID)
	if err != nil {
		t.Fatalf("Get job: %v", err)
	}
	if got.RepeatCompleted != len(runs) {
		t.Fatalf("RepeatCompleted = %d, want %d with no lost parallel completions", got.RepeatCompleted, len(runs))
	}
	if got.LastRunUnix != start+3 {
		t.Fatalf("LastRunUnix = %d, want latest started_at %d", got.LastRunUnix, start+3)
	}
	if got.LastStatus != "suppressed" {
		t.Fatalf("LastStatus = %q, want latest run status suppressed", got.LastStatus)
	}

	persisted, err := e.cfg.RunStore.LatestRuns(context.Background(), job.ID, 10)
	if err != nil {
		t.Fatalf("LatestRuns: %v", err)
	}
	if len(persisted) != len(runs) {
		t.Fatalf("persisted runs = %d, want %d", len(persisted), len(runs))
	}
}

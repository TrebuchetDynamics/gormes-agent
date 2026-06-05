package kanban

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestKanbanRunHistoryRecordsSpawn(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()
	store.now = func() time.Time { return now }

	task, err := store.CreateTask(ctx, CreateTaskInput{
		Title:    "Spawn test task",
		Assignee: "test-worker",
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	// Claim the task to set it to running.
	if _, claimed, err := store.ClaimTask(ctx, task.ID, ClaimTaskInput{
		Worker: "test-worker",
		TTL:    15 * time.Minute,
	}); err != nil || !claimed {
		t.Fatalf("ClaimTask() = claimed %v, err %v, want claimed=true, err=nil", claimed, err)
	}

	// Record a successful spawn.
	if err := store.recordSpawned(ctx, task.ID, ProcessStartResult{
		PID:       4242,
		StartedAt: now,
	}); err != nil {
		t.Fatalf("recordSpawned() error = %v", err)
	}

	// Verify the run history contains the spawn record.
	runs, err := store.ListRuns(ctx, task.ID)
	if err != nil {
		t.Fatalf("ListRuns() error = %v", err)
	}
	if len(runs) == 0 {
		t.Fatal("ListRuns() returned 0 runs, want at least 1")
	}
	var spawnRun *TaskRun
	for i := range runs {
		if runs[i].Outcome == RunOutcomeSpawned {
			spawnRun = &runs[i]
			break
		}
	}
	if spawnRun == nil {
		t.Fatalf("no run with outcome %q found in %d runs", RunOutcomeSpawned, len(runs))
	}
	if spawnRun.StartedAt.IsZero() {
		t.Errorf("spawn run started_at is zero, want non-zero timestamp")
	}
}

func TestKanbanRunHistoryRecordsCompletion(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()
	store.now = func() time.Time { return now }

	task, err := store.CreateTask(ctx, CreateTaskInput{
		Title:    "Completion test task",
		Assignee: "test-worker",
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	// Claim the task to set it to running.
	if _, claimed, err := store.ClaimTask(ctx, task.ID, ClaimTaskInput{
		Worker: "test-worker",
		TTL:    15 * time.Minute,
	}); err != nil || !claimed {
		t.Fatalf("ClaimTask() = claimed %v, err %v, want claimed=true, err=nil", claimed, err)
	}

	// Complete the task.
	if err := store.CompleteTask(ctx, task.ID, CompleteTaskInput{
		Result: "task done",
	}); err != nil {
		t.Fatalf("CompleteTask() error = %v", err)
	}

	// Verify the run history contains the completion record.
	runs, err := store.ListRuns(ctx, task.ID)
	if err != nil {
		t.Fatalf("ListRuns() error = %v", err)
	}
	if len(runs) == 0 {
		t.Fatal("ListRuns() returned 0 runs, want at least 1")
	}
	var completeRun *TaskRun
	for i := range runs {
		if runs[i].Outcome == RunOutcomeCompleted {
			completeRun = &runs[i]
			break
		}
	}
	if completeRun == nil {
		t.Fatalf("no run with outcome %q found in %d runs", RunOutcomeCompleted, len(runs))
	}
	if completeRun.EndedAt.IsZero() {
		t.Errorf("completion run ended_at is zero, want non-zero timestamp")
	}
}

func TestKanbanRunHistoryAutoBlockAfterConsecutiveFailures(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()
	store.now = func() time.Time { return now }

	task, err := store.CreateTask(ctx, CreateTaskInput{
		Title:    "Auto-block test task",
		Assignee: "test-worker",
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	// Need to claim each time because releaseFailedSpawn updates the task status.
	for i := range 5 {
		store.now = func() time.Time { return now.Add(time.Duration(i) * time.Second) }

		if _, claimed, err := store.ClaimTask(ctx, task.ID, ClaimTaskInput{
			Worker: "test-worker",
			TTL:    15 * time.Minute,
		}); err != nil {
			t.Fatalf("ClaimTask() iteration %d error = %v", i, err)
		} else if !claimed {
			t.Fatalf("ClaimTask() iteration %d not claimed, want claimed", i)
		}

		blocked, err := store.releaseFailedSpawn(ctx, task.ID, "test failure", 5)
		if err != nil {
			t.Fatalf("releaseFailedSpawn() iteration %d error = %v", i, err)
		}
		if i == 4 && !blocked {
			t.Errorf("releaseFailedSpawn() iteration %d reported blocked=false, want blocked=true (after 5th consecutive failure)", i)
		}
	}

	got, err := store.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if got.Status != StatusBlocked {
		t.Errorf("task status = %q, want %q after 5 consecutive spawn failures", got.Status, StatusBlocked)
	}
}

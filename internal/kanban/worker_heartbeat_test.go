package kanban

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestKanbanWorkerHeartbeatExtendsClaimExpiry(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()
	store.now = func() time.Time { return now }

	task, err := store.CreateTask(ctx, CreateTaskInput{
		Title:         "Heartbeat task",
		Assignee:      "researcher",
		WorkspaceKind: WorkspaceScratch,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	claimTTL := 2 * time.Minute
	if _, claimed, err := store.ClaimTask(ctx, task.ID, ClaimTaskInput{Worker: "w1", TTL: claimTTL}); err != nil || !claimed {
		t.Fatalf("ClaimTask() = claimed %v err %v, want claimed", claimed, err)
	}

	claimed, err := store.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	initialExpiry := claimed.ClaimExpires

	// Advance time by 1 minute (still within 2 min claim TTL).
	store.now = func() time.Time { return now.Add(1 * time.Minute) }

	heartbeatTTL := 3 * time.Minute
	ok, err := store.HeartbeatTask(ctx, task.ID, heartbeatTTL, "still working")
	if err != nil {
		t.Fatalf("HeartbeatTask() error = %v", err)
	}
	if !ok {
		t.Fatalf("HeartbeatTask() = false, want true")
	}

	refreshed, err := store.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask() after heartbeat error = %v", err)
	}
	if !refreshed.ClaimExpires.After(initialExpiry) {
		t.Fatalf("ClaimExpires after heartbeat = %v, want after initial %v", refreshed.ClaimExpires, initialExpiry)
	}
	// Heartbeat should extend by heartbeatTTL from "now" (10:01:00).
	expectedExpiry := now.Add(1*time.Minute).Add(heartbeatTTL)
	if refreshed.ClaimExpires.Sub(expectedExpiry).Abs() > time.Second {
		t.Fatalf("ClaimExpires = %v, want ~%v", refreshed.ClaimExpires, expectedExpiry)
	}

	// Verify heartbeat event recorded.
	events, err := store.ListEvents(ctx, task.ID)
	if err != nil {
		t.Fatalf("ListEvents() error = %v", err)
	}
	found := false
	for _, ev := range events {
		if ev.Kind == "heartbeat" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("heartbeat event not recorded")
	}
}

func TestKanbanWorkerHeartbeatFailsOnNonRunningTask(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()
	store.now = func() time.Time { return now }

	task, err := store.CreateTask(ctx, CreateTaskInput{
		Title:         "Non-running heartbeat",
		Assignee:      "researcher",
		WorkspaceKind: WorkspaceScratch,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	ok, err := store.HeartbeatTask(ctx, task.ID, 3*time.Minute, "test")
	if err != nil {
		t.Fatalf("HeartbeatTask() error = %v", err)
	}
	if ok {
		t.Fatalf("HeartbeatTask() on ready task returned true, want false")
	}
}

func TestKanbanWorkerHeartbeatStaleReclaim(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()
	store.now = func() time.Time { return now }

	task, err := store.CreateTask(ctx, CreateTaskInput{
		Title:         "Stale heartbeat task",
		Assignee:      "researcher",
		WorkspaceKind: WorkspaceScratch,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	claimTTL := 15 * time.Minute
	if _, claimed, err := store.ClaimTask(ctx, task.ID, ClaimTaskInput{Worker: "stale-worker", TTL: claimTTL}); err != nil || !claimed {
		t.Fatalf("ClaimTask() = claimed %v err %v, want claimed", claimed, err)
	}

	// Heartbeat once at start.
	heartbeatTTL := 60 * time.Second
	if _, err := store.HeartbeatTask(ctx, task.ID, heartbeatTTL, "started"); err != nil {
		t.Fatalf("HeartbeatTask() error = %v", err)
	}

	// Advance past heartbeat TTL but within claim TTL.
	store.now = func() time.Time { return now.Add(2 * heartbeatTTL) }

	// Reclaim should detect stale heartbeat and mark worker as zombie.
	heartbeatTimeout := heartbeatTTL
	reclaimed, err := store.ReclaimStaleHeartbeats(ctx, heartbeatTimeout)
	if err != nil {
		t.Fatalf("ReclaimStaleHeartbeats() error = %v", err)
	}
	if len(reclaimed) != 1 || reclaimed[0] != task.ID {
		t.Fatalf("ReclaimedIDs = %v, want [%s]", reclaimed, task.ID)
	}

	reclaimedTask, err := store.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if reclaimedTask.Status != StatusReady {
		t.Fatalf("reclaimed task status = %q, want %q", reclaimedTask.Status, StatusReady)
	}

	// Verify zombie event recorded.
	events, err := store.ListEvents(ctx, task.ID)
	if err != nil {
		t.Fatalf("ListEvents() error = %v", err)
	}
	foundZombie := false
	for _, ev := range events {
		if ev.Kind == "worker_zombie" {
			foundZombie = true
			break
		}
	}
	if !foundZombie {
		t.Fatalf("worker_zombie event not recorded")
	}

	// Retry budget should be decremented exactly once.
	runs, err := store.ListRuns(ctx, task.ID)
	if err != nil {
		t.Fatalf("ListRuns() error = %v", err)
	}
	zombieCount := 0
	for _, run := range runs {
		if run.Outcome == RunOutcomeWorkerZombie {
			zombieCount++
		}
	}
	if zombieCount != 1 {
		t.Fatalf("zombie run count = %d, want 1", zombieCount)
	}
}

func TestKanbanWorkerHeartbeatHealthyDoesNotReclaim(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()
	store.now = func() time.Time { return now }

	task, err := store.CreateTask(ctx, CreateTaskInput{
		Title:         "Healthy heartbeat task",
		Assignee:      "researcher",
		WorkspaceKind: WorkspaceScratch,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	if _, claimed, err := store.ClaimTask(ctx, task.ID, ClaimTaskInput{Worker: "healthy-worker", TTL: 15 * time.Minute}); err != nil || !claimed {
		t.Fatalf("ClaimTask() = claimed %v err %v, want claimed", claimed, err)
	}

	heartbeatTTL := 60 * time.Second
	if _, err := store.HeartbeatTask(ctx, task.ID, heartbeatTTL, "alive"); err != nil {
		t.Fatalf("HeartbeatTask() error = %v", err)
	}

	// Advance only 30s (still within heartbeat TTL).
	store.now = func() time.Time { return now.Add(30 * time.Second) }

	reclaimed, err := store.ReclaimStaleHeartbeats(ctx, heartbeatTTL)
	if err != nil {
		t.Fatalf("ReclaimStaleHeartbeats() error = %v", err)
	}
	if len(reclaimed) != 0 {
		t.Fatalf("ReclaimedIDs = %v, want empty (healthy worker not reclaimed)", reclaimed)
	}
}

func TestKanbanWorkerFailureCounterUnified(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()
	store.now = func() time.Time { return now }

	task, err := store.CreateTask(ctx, CreateTaskInput{
		Title:         "Failure counter task",
		Assignee:      "researcher",
		WorkspaceKind: WorkspaceScratch,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	// Spawn-failure increments counter.
	if err := store.IncrementFailureCounter(ctx, task.ID, FailureKindSpawn); err != nil {
		t.Fatalf("IncrementFailureCounter(spawn) error = %v", err)
	}
	got, _ := store.GetTask(ctx, task.ID)
	fc1 := got.FailureCount()
	if fc1 != 1 {
		t.Fatalf("FailureCount after spawn = %d, want 1", fc1)
	}

	// Timeout increments same counter.
	if err := store.IncrementFailureCounter(ctx, task.ID, FailureKindTimeout); err != nil {
		t.Fatalf("IncrementFailureCounter(timeout) error = %v", err)
	}
	got, _ = store.GetTask(ctx, task.ID)
	fc2 := got.FailureCount()
	if fc2 != 2 {
		t.Fatalf("FailureCount after spawn+timeout = %d, want 2", fc2)
	}

	// Crash increments same counter.
	if err := store.IncrementFailureCounter(ctx, task.ID, FailureKindCrash); err != nil {
		t.Fatalf("IncrementFailureCounter(crash) error = %v", err)
	}
	got, _ = store.GetTask(ctx, task.ID)
	fc3 := got.FailureCount()
	if fc3 != 3 {
		t.Fatalf("FailureCount after spawn+timeout+crash = %d, want 3", fc3)
	}

	// Verify failure counter visible in dispatcher status.
	events, err := store.ListEvents(ctx, task.ID)
	if err != nil {
		t.Fatalf("ListEvents() error = %v", err)
	}
	failureKinds := map[string]int{}
	for _, ev := range events {
		if ev.Kind == "spawn_failed" || ev.Kind == "worker_timed_out" || ev.Kind == "worker_crashed" {
			failureKinds[ev.Kind]++
		}
	}
	if len(failureKinds) != 3 {
		t.Fatalf("failure event kinds = %d, want 3", len(failureKinds))
	}
}

func TestKanbanWorkerAutoBlockOnIncompleteExit(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()
	store.now = func() time.Time { return now }

	task, err := store.CreateTask(ctx, CreateTaskInput{
		Title:         "Auto-block task",
		Assignee:      "researcher",
		WorkspaceKind: WorkspaceScratch,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	if _, claimed, err := store.ClaimTask(ctx, task.ID, ClaimTaskInput{Worker: "exiting-worker", TTL: 15 * time.Minute}); err != nil || !claimed {
		t.Fatalf("ClaimTask() = claimed %v err %v, want claimed", claimed, err)
	}

	// Auto-block on incomplete exit.
	if err := store.AutoBlockIncompleteExit(ctx, task.ID, "worker exited without completing"); err != nil {
		t.Fatalf("AutoBlockIncompleteExit() error = %v", err)
	}

	blocked, err := store.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if blocked.Status != StatusBlocked {
		t.Fatalf("blocked task status = %q, want %q", blocked.Status, StatusBlocked)
	}

	// Verify auto_blocked event recorded.
	events, err := store.ListEvents(ctx, task.ID)
	if err != nil {
		t.Fatalf("ListEvents() error = %v", err)
	}
	found := false
	for _, ev := range events {
		if ev.Kind == "auto_blocked" {
			found = true
			if ev.Payload == "" {
				t.Fatalf("auto_blocked event has empty payload")
			}
			break
		}
	}
	if !found {
		t.Fatalf("auto_blocked event not recorded")
	}
}

func TestKanbanWorkerDarwinZombieDetected(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()
	store.now = func() time.Time { return now }

	task, err := store.CreateTask(ctx, CreateTaskInput{
		Title:         "Darwin zombie task",
		Assignee:      "researcher",
		WorkspaceKind: WorkspaceScratch,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	if _, claimed, err := store.ClaimTask(ctx, task.ID, ClaimTaskInput{Worker: "darwin-worker", TTL: 15 * time.Minute}); err != nil || !claimed {
		t.Fatalf("ClaimTask() = claimed %v err %v, want claimed", claimed, err)
	}

	zombieDetected := isWorkerProcessZombie(99999)

	if runtime.GOOS == "darwin" {
		// On darwin, PID 99999 should not exist, so it's a zombie (process gone).
		if !zombieDetected {
			t.Fatalf("isWorkerProcessZombie(99999) on darwin = false, want true")
		}
	} else {
		// On other platforms, returns false (unknown detection path).
		// Must not panic.
		_ = zombieDetected
	}
}

func TestKanbanWorkerDarwinZombieDetectionDoesNotPanic(t *testing.T) {
	// Verify that calling isWorkerProcessZombie on any GOOS does not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("isWorkerProcessZombie panicked: %v", r)
		}
	}()
	for _, pid := range []int{0, 1, 99999, -1} {
		_ = isWorkerProcessZombie(pid)
	}
}

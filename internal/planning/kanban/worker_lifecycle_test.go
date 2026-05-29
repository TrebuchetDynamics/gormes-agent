package kanban

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestKanbanWorkerLifecycleReclaimsCrashedAndTimedOutWorkers(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 6, 19, 0, 0, 0, time.UTC)
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	crashedStarted := now.Add(-30 * time.Minute)
	crashed := createRunningWorkerTask(t, ctx, store, crashedStarted, "crashed worker", 111)
	timedOutStarted := now.Add(-2 * time.Hour)
	timedOut := createRunningWorkerTask(t, ctx, store, timedOutStarted, "timed out worker", 222)
	reusedStarted := now.Add(-2 * time.Hour)
	reused := createRunningWorkerTask(t, ctx, store, reusedStarted, "reused pid worker", 333)
	store.now = func() time.Time { return now }

	processes := &fakeWorkerProcessController{
		snapshots: map[int]ProcessSnapshot{
			111: {PID: 111, Live: false, StartedAt: crashedStarted},
			222: {PID: 222, Live: true, StartedAt: timedOutStarted},
			333: {PID: 333, Live: true, StartedAt: now.Add(-time.Minute)},
		},
	}
	monitor := WorkerLifecycleMonitor{
		Store:      store,
		Processes:  processes,
		Now:        func() time.Time { return now },
		MaxRuntime: time.Hour,
	}

	crashedIDs, err := monitor.DetectCrashedWorkers(ctx)
	if err != nil {
		t.Fatalf("DetectCrashedWorkers() error = %v", err)
	}
	if !reflect.DeepEqual(crashedIDs, []string{crashed.ID}) {
		t.Fatalf("crashed IDs = %v, want [%s]", crashedIDs, crashed.ID)
	}
	timedOutIDs, err := monitor.EnforceMaxRuntime(ctx)
	if err != nil {
		t.Fatalf("EnforceMaxRuntime() error = %v", err)
	}
	if !reflect.DeepEqual(timedOutIDs, []string{timedOut.ID}) {
		t.Fatalf("timed out IDs = %v, want [%s]", timedOutIDs, timedOut.ID)
	}
	if !reflect.DeepEqual(processes.stopped, []int{222}) {
		t.Fatalf("stopped PIDs = %v, want [222]", processes.stopped)
	}

	assertTaskStatus(t, ctx, store, crashed.ID, StatusReady)
	assertTaskStatus(t, ctx, store, timedOut.ID, StatusReady)
	assertTaskStatus(t, ctx, store, reused.ID, StatusRunning)
	assertLastRunOutcome(t, ctx, store, crashed.ID, RunOutcomeWorkerCrashed)
	assertLastRunOutcome(t, ctx, store, timedOut.ID, RunOutcomeWorkerTimedOut)
}

func createRunningWorkerTask(t *testing.T, ctx context.Context, store *Store, startedAt time.Time, title string, pid int) Task {
	t.Helper()
	store.now = func() time.Time { return startedAt }
	task, err := store.CreateTask(ctx, CreateTaskInput{Title: title, Assignee: "coder"})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if _, claimed, err := store.ClaimTask(ctx, task.ID, ClaimTaskInput{Worker: "kanban-dispatcher", TTL: time.Hour}); err != nil || !claimed {
		t.Fatalf("ClaimTask() claimed=%v err=%v, want claimed", claimed, err)
	}
	if err := store.recordSpawned(ctx, task.ID, ProcessStartResult{PID: pid, StartedAt: startedAt}); err != nil {
		t.Fatalf("recordSpawned() error = %v", err)
	}
	task, err = store.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	return task
}

func assertTaskStatus(t *testing.T, ctx context.Context, store *Store, id string, want Status) {
	t.Helper()
	task, err := store.GetTask(ctx, id)
	if err != nil {
		t.Fatalf("GetTask(%s) error = %v", id, err)
	}
	if task.Status != want {
		t.Fatalf("task %s status = %q, want %q", id, task.Status, want)
	}
}

func assertLastRunOutcome(t *testing.T, ctx context.Context, store *Store, id string, want RunOutcome) {
	t.Helper()
	runs, err := store.ListRuns(ctx, id)
	if err != nil {
		t.Fatalf("ListRuns(%s) error = %v", id, err)
	}
	if len(runs) == 0 {
		t.Fatalf("ListRuns(%s) returned no runs", id)
	}
	if got := runs[len(runs)-1].Outcome; got != want {
		t.Fatalf("last run outcome for %s = %q, want %q", id, got, want)
	}
}

type fakeWorkerProcessController struct {
	snapshots map[int]ProcessSnapshot
	stopped   []int
}

func (f *fakeWorkerProcessController) Snapshot(_ context.Context, pid int) (ProcessSnapshot, error) {
	return f.snapshots[pid], nil
}

func (f *fakeWorkerProcessController) Stop(_ context.Context, pid int, _ time.Time) error {
	f.stopped = append(f.stopped, pid)
	return nil
}

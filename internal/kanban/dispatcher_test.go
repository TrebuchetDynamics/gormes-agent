package kanban

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestKanbanDispatcherReclaimsStaleAndSpawnsReadyWithMaxCap(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 4, 16, 0, 0, 0, time.UTC)
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()
	store.now = func() time.Time { return now }

	stale, err := store.CreateTask(ctx, CreateTaskInput{
		Title:         "Recover stale worker task",
		Assignee:      "researcher",
		WorkspaceKind: WorkspaceScratch,
	})
	if err != nil {
		t.Fatalf("CreateTask(stale) error = %v", err)
	}
	store.now = func() time.Time { return now.Add(time.Second) }
	next, err := store.CreateTask(ctx, CreateTaskInput{
		Title:         "Leave queued behind max cap",
		Assignee:      "backend-dev",
		WorkspaceKind: WorkspaceScratch,
	})
	if err != nil {
		t.Fatalf("CreateTask(next) error = %v", err)
	}
	if _, claimed, err := store.ClaimTask(ctx, stale.ID, ClaimTaskInput{Worker: "old-worker", TTL: time.Minute}); err != nil || !claimed {
		t.Fatalf("ClaimTask(stale) = claimed %v err %v, want claimed", claimed, err)
	}

	store.now = func() time.Time { return now.Add(3 * time.Minute) }
	var requests []SpawnRequest
	dispatcher := Dispatcher{
		Store: store,
		Spawner: SpawnFunc(func(_ context.Context, req SpawnRequest) (SpawnResult, error) {
			requests = append(requests, req)
			return SpawnResult{PID: 4242}, nil
		}),
	}

	result, err := dispatcher.RunOnce(ctx, DispatchOptions{MaxSpawn: 1})
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if len(result.ReclaimedIDs) != 1 || result.ReclaimedIDs[0] != stale.ID {
		t.Fatalf("ReclaimedIDs = %v, want [%s]", result.ReclaimedIDs, stale.ID)
	}
	if len(result.Spawned) != 1 || result.Spawned[0].TaskID != stale.ID {
		t.Fatalf("Spawned = %+v, want one stale task spawn", result.Spawned)
	}
	if len(requests) != 1 {
		t.Fatalf("spawn requests = %d, want 1", len(requests))
	}
	req := requests[0]
	if req.Task.ID != stale.ID || req.Task.Assignee != "researcher" {
		t.Fatalf("spawn request task = %+v, want stale researcher task", req.Task)
	}
	if req.Env["GORMES_KANBAN_TASK"] != stale.ID {
		t.Fatalf("GORMES_KANBAN_TASK = %q, want %q", req.Env["GORMES_KANBAN_TASK"], stale.ID)
	}
	if req.Env["GORMES_PROFILE"] != "researcher" {
		t.Fatalf("GORMES_PROFILE = %q, want researcher", req.Env["GORMES_PROFILE"])
	}
	if _, ok := req.Env["HERMES_KANBAN_TASK"]; ok {
		t.Fatalf("spawn env leaked Hermes key: %+v", req.Env)
	}

	staleAfter, err := store.GetTask(ctx, stale.ID)
	if err != nil {
		t.Fatalf("GetTask(stale) error = %v", err)
	}
	if staleAfter.Status != StatusRunning || staleAfter.ClaimLock == "" || staleAfter.ClaimExpires.IsZero() {
		t.Fatalf("stale after dispatch = %+v, want running claimed task", staleAfter)
	}
	nextAfter, err := store.GetTask(ctx, next.ID)
	if err != nil {
		t.Fatalf("GetTask(next) error = %v", err)
	}
	if nextAfter.Status != StatusReady {
		t.Fatalf("next task Status = %q, want %q because max cap held it", nextAfter.Status, StatusReady)
	}
}

func TestKanbanDispatcherNamedBoardWorkspaceRoot(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	dbPath := filepath.Join(root, "kanban", "boards", "alpha", "kanban.db")
	store, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	task, err := store.CreateTask(ctx, CreateTaskInput{
		Title:         "Named board scratch workspace",
		Assignee:      "worker",
		WorkspaceKind: WorkspaceScratch,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	var requests []SpawnRequest
	dispatcher := Dispatcher{
		Store: store,
		Spawner: SpawnFunc(func(_ context.Context, req SpawnRequest) (SpawnResult, error) {
			requests = append(requests, req)
			return SpawnResult{PID: 1234}, nil
		}),
	}
	if _, err := dispatcher.RunOnce(ctx, DispatchOptions{MaxSpawn: 1}); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("spawn requests = %d, want 1", len(requests))
	}
	want := filepath.Join(root, "kanban", "boards", "alpha", "workspaces", task.ID)
	if requests[0].WorkspacePath != want {
		t.Fatalf("WorkspacePath = %q, want %q", requests[0].WorkspacePath, want)
	}
	if requests[0].Env["GORMES_KANBAN_WORKSPACE"] != want {
		t.Fatalf("GORMES_KANBAN_WORKSPACE = %q, want %q", requests[0].Env["GORMES_KANBAN_WORKSPACE"], want)
	}
	if info, err := os.Stat(want); err != nil {
		t.Fatalf("workspace dir missing: %v", err)
	} else if !info.IsDir() {
		t.Fatalf("workspace path is not a directory: %s", want)
	}
}

func TestKanbanDispatcherDefaultBoardWorkspaceRootPreservesLegacy(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store, err := Open(ctx, filepath.Join(root, "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	task, err := store.CreateTask(ctx, CreateTaskInput{
		Title:         "Default board scratch workspace",
		Assignee:      "worker",
		WorkspaceKind: WorkspaceScratch,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	var requests []SpawnRequest
	dispatcher := Dispatcher{
		Store: store,
		Spawner: SpawnFunc(func(_ context.Context, req SpawnRequest) (SpawnResult, error) {
			requests = append(requests, req)
			return SpawnResult{PID: 1234}, nil
		}),
	}
	if _, err := dispatcher.RunOnce(ctx, DispatchOptions{MaxSpawn: 1}); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("spawn requests = %d, want 1", len(requests))
	}
	want := filepath.Join(root, "kanban", "workspaces", task.ID)
	if requests[0].WorkspacePath != want {
		t.Fatalf("WorkspacePath = %q, want %q", requests[0].WorkspacePath, want)
	}
}

func TestKanbanDispatcherBlocksTaskAfterSpawnFailureLimit(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	task, err := store.CreateTask(ctx, CreateTaskInput{
		Title:    "Needs missing profile",
		Assignee: "missing-profile",
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	dispatcher := Dispatcher{
		Store: store,
		Spawner: SpawnFunc(func(context.Context, SpawnRequest) (SpawnResult, error) {
			return SpawnResult{}, errors.New("profile missing")
		}),
	}

	first, err := dispatcher.RunOnce(ctx, DispatchOptions{FailureLimit: 2})
	if err != nil {
		t.Fatalf("first RunOnce() error = %v", err)
	}
	if len(first.SpawnFailedIDs) != 1 || first.SpawnFailedIDs[0] != task.ID {
		t.Fatalf("first SpawnFailedIDs = %v, want [%s]", first.SpawnFailedIDs, task.ID)
	}
	if len(first.AutoBlockedTaskIDs) != 0 {
		t.Fatalf("first AutoBlockedTaskIDs = %v, want none before limit", first.AutoBlockedTaskIDs)
	}
	afterFirst, err := store.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask(after first) error = %v", err)
	}
	if afterFirst.Status != StatusReady {
		t.Fatalf("after first Status = %q, want retryable %q", afterFirst.Status, StatusReady)
	}

	second, err := dispatcher.RunOnce(ctx, DispatchOptions{FailureLimit: 2})
	if err != nil {
		t.Fatalf("second RunOnce() error = %v", err)
	}
	if len(second.AutoBlockedTaskIDs) != 1 || second.AutoBlockedTaskIDs[0] != task.ID {
		t.Fatalf("second AutoBlockedTaskIDs = %v, want [%s]", second.AutoBlockedTaskIDs, task.ID)
	}
	afterSecond, err := store.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask(after second) error = %v", err)
	}
	if afterSecond.Status != StatusBlocked {
		t.Fatalf("after second Status = %q, want %q", afterSecond.Status, StatusBlocked)
	}
	if afterSecond.Result == "" {
		t.Fatal("blocked task Result is empty, want failure reason evidence")
	}

	runs, err := store.ListRuns(ctx, task.ID)
	if err != nil {
		t.Fatalf("ListRuns() error = %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("runs = %+v, want two recorded spawn attempts", runs)
	}
	if runs[0].Outcome != RunOutcomeSpawnFailed || runs[1].Outcome != RunOutcomeGaveUp {
		t.Fatalf("run outcomes = %q, %q; want %q, %q", runs[0].Outcome, runs[1].Outcome, RunOutcomeSpawnFailed, RunOutcomeGaveUp)
	}
}

func TestKanbanDispatcherMaxSpawnCapsFailedAttempts(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 4, 16, 30, 0, 0, time.UTC)
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()
	store.now = func() time.Time { return now }

	broken, err := store.CreateTask(ctx, CreateTaskInput{
		Title:    "Broken profile first",
		Assignee: "broken-profile",
	})
	if err != nil {
		t.Fatalf("CreateTask(broken) error = %v", err)
	}
	store.now = func() time.Time { return now.Add(time.Second) }
	next, err := store.CreateTask(ctx, CreateTaskInput{
		Title:    "Must wait for next tick",
		Assignee: "backend-dev",
	})
	if err != nil {
		t.Fatalf("CreateTask(next) error = %v", err)
	}

	var attempted []string
	dispatcher := Dispatcher{
		Store: store,
		Spawner: SpawnFunc(func(_ context.Context, req SpawnRequest) (SpawnResult, error) {
			attempted = append(attempted, req.Task.ID)
			return SpawnResult{}, errors.New("spawn refused")
		}),
	}

	result, err := dispatcher.RunOnce(ctx, DispatchOptions{MaxSpawn: 1})
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if len(result.SpawnFailedIDs) != 1 || result.SpawnFailedIDs[0] != broken.ID {
		t.Fatalf("SpawnFailedIDs = %v, want [%s]", result.SpawnFailedIDs, broken.ID)
	}
	if len(attempted) != 1 || attempted[0] != broken.ID {
		t.Fatalf("spawn attempts = %v, want only failed first task", attempted)
	}
	nextAfter, err := store.GetTask(ctx, next.ID)
	if err != nil {
		t.Fatalf("GetTask(next) error = %v", err)
	}
	if nextAfter.Status != StatusReady {
		t.Fatalf("next task Status = %q, want %q because failed attempt consumed MaxSpawn", nextAfter.Status, StatusReady)
	}
}

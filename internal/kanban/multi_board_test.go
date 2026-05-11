package kanban

import (
	"context"
	"path/filepath"
	"testing"
)

func TestKanbanMultiBoardIsolation(t *testing.T) {
	ctx := context.Background()

	// Create two independent board stores.
	alphaStore, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open(alpha) error = %v", err)
	}
	defer alphaStore.Close()

	betaStore, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open(beta) error = %v", err)
	}
	defer betaStore.Close()

	// Create a task in board_alpha.
	task, err := alphaStore.CreateTask(ctx, CreateTaskInput{
		Title:    "Board alpha task",
		Assignee: "worker-alpha",
	})
	if err != nil {
		t.Fatalf("CreateTask(alpha) error = %v", err)
	}
	if task.Status != StatusReady {
		t.Fatalf("task.Status = %q, want %q", task.Status, StatusReady)
	}

	// Verify board_beta cannot query the alpha task.
	_, err = betaStore.GetTask(ctx, task.ID)
	if err == nil {
		t.Fatal("betaStore.GetTask(alpha task) expected error, got nil")
	}

	// Verify board_beta's task list is empty.
	betaTasks, err := betaStore.ListTasks(ctx, ListFilter{})
	if err != nil {
		t.Fatalf("betaStore.ListTasks() error = %v", err)
	}
	if len(betaTasks) != 0 {
		t.Fatalf("betaStore.ListTasks() = %d tasks, want 0", len(betaTasks))
	}

	// Verify board_alpha itself can still see the task.
	alphaTasks, err := alphaStore.ListTasks(ctx, ListFilter{})
	if err != nil {
		t.Fatalf("alphaStore.ListTasks() error = %v", err)
	}
	if len(alphaTasks) != 1 {
		t.Fatalf("alphaStore.ListTasks() = %d tasks, want 1", len(alphaTasks))
	}
}

func TestKanbanMultiBoardDispatcherHonoursBoardBoundary(t *testing.T) {
	ctx := context.Background()

	// Create a dispatcher for board_alpha with a task.
	alphaStore, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open(alpha) error = %v", err)
	}
	defer alphaStore.Close()

	task, err := alphaStore.CreateTask(ctx, CreateTaskInput{
		Title:         "Alpha task",
		Assignee:      "worker",
		WorkspaceKind: WorkspaceScratch,
	})
	if err != nil {
		t.Fatalf("CreateTask(alpha) error = %v", err)
	}

	// Create a separate store for board_beta.
	betaStore, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open(beta) error = %v", err)
	}
	defer betaStore.Close()

	// The dispatcher for board_beta should not see tasks from board_alpha.
	// ValidateTaskBoard proves cross-board rejection.
	if err := ValidateTaskBoard(ctx, betaStore, task.ID); err == nil {
		t.Fatal("ValidateTaskBoard(beta, alpha-task) expected ErrCrossBoardTask, got nil")
	}

	// ValidateTaskBoard for alpha's own store succeeds.
	if err := ValidateTaskBoard(ctx, alphaStore, task.ID); err != nil {
		t.Fatalf("ValidateTaskBoard(alpha, alpha-task) error = %v, want nil", err)
	}

	// BoardDispatcher for board_beta should have no ready tasks.
	var spawnRequests int
	betaDispatcher, err := NewBoardDispatcher(ctx, Board{
		Name: "board_beta",
		Path: betaStore.DBPath(),
	}, SpawnFunc(func(_ context.Context, _ SpawnRequest) (SpawnResult, error) {
		spawnRequests++
		return SpawnResult{PID: 123}, nil
	}))
	if err != nil {
		t.Fatalf("NewBoardDispatcher(beta) error = %v", err)
	}
	defer betaDispatcher.Close()

	result, err := betaDispatcher.RunOnce(ctx, DispatchOptions{MaxSpawn: 5})
	if err != nil {
		t.Fatalf("betaDispatcher.RunOnce() error = %v", err)
	}
	if len(result.Spawned) != 0 {
		t.Fatalf("betaDispatcher.RunOnce() spawned %d tasks, want 0 (no tasks on board_beta)", len(result.Spawned))
	}
	if spawnRequests != 0 {
		t.Fatalf("betaDispatcher issued %d spawn requests, want 0", spawnRequests)
	}
}

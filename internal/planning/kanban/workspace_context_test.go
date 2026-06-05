package kanban

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestKanbanWorkspaceContextInjection(t *testing.T) {
	wsDir := t.TempDir()
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { store.Close() })

	task, err := store.CreateTask(ctx, CreateTaskInput{
		Title:         "test workspace task",
		Assignee:      "worker-a",
		WorkspaceKind: WorkspaceDir,
		WorkspacePath: wsDir,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	resolved, err := store.resolveWorkspace(ctx, task)
	if err != nil {
		t.Fatalf("resolveWorkspace() error = %v", err)
	}
	if resolved != wsDir {
		t.Fatalf("resolveWorkspace() = %q, want %q", resolved, wsDir)
	}
}

func TestKanbanWorkspaceContextInjectionScratch(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { store.Close() })

	task, err := store.CreateTask(ctx, CreateTaskInput{
		Title:         "scratch workspace task",
		Assignee:      "worker-b",
		WorkspaceKind: WorkspaceScratch,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	resolved, err := store.resolveWorkspace(ctx, task)
	if err != nil {
		t.Fatalf("resolveWorkspace() error = %v", err)
	}
	if resolved == "" {
		t.Fatalf("resolveWorkspace() = empty string, want scratch path")
	}
	if _, err := os.Stat(resolved); err != nil {
		t.Fatalf("scratch workspace dir %q does not exist: %v", resolved, err)
	}
}

func TestKanbanWorkspaceAGENTSLoad(t *testing.T) {
	wsDir := t.TempDir()
	agentsContent := "# Agent Instructions\n\nBe helpful and concise."
	if err := os.WriteFile(filepath.Join(wsDir, "AGENTS.md"), []byte(agentsContent), 0o644); err != nil {
		t.Fatalf("WriteFile(AGENTS.md) error = %v", err)
	}

	ctx, err := LoadWorkspaceContext(wsDir)
	if err != nil {
		t.Fatalf("LoadWorkspaceContext() error = %v", err)
	}
	if ctx.WorkspaceDir != wsDir {
		t.Fatalf("WorkspaceDir = %q, want %q", ctx.WorkspaceDir, wsDir)
	}
	if ctx.AGENTSMD != agentsContent {
		t.Fatalf("AGENTSMD = %q, want %q", ctx.AGENTSMD, agentsContent)
	}
}

func TestKanbanWorkspaceAGENTSLoadAbsent(t *testing.T) {
	wsDir := t.TempDir()

	ctx, err := LoadWorkspaceContext(wsDir)
	if err != nil {
		t.Fatalf("LoadWorkspaceContext() error = %v", err)
	}
	if ctx.WorkspaceDir != wsDir {
		t.Fatalf("WorkspaceDir = %q, want %q", ctx.WorkspaceDir, wsDir)
	}
	if ctx.AGENTSMD != "" {
		t.Fatalf("AGENTSMD = %q, want empty string for absent file", ctx.AGENTSMD)
	}
}

func TestKanbanWorkspaceAGENTSLoadNonExistentDir(t *testing.T) {
	wsDir := filepath.Join(t.TempDir(), "nonexistent")

	ctx, err := LoadWorkspaceContext(wsDir)
	if err != nil {
		t.Fatalf("LoadWorkspaceContext() error = %v", err)
	}
	if ctx.WorkspaceDir != wsDir {
		t.Fatalf("WorkspaceDir = %q, want %q", ctx.WorkspaceDir, wsDir)
	}
	if ctx.AGENTSMD != "" {
		t.Fatalf("AGENTSMD = %q, want empty string for nonexistent dir", ctx.AGENTSMD)
	}
}

func TestKanbanWorkspaceAGENTSLoadEmptyDir(t *testing.T) {
	wsDir := t.TempDir()

	ctx, err := LoadWorkspaceContext(wsDir)
	if err != nil {
		t.Fatalf("LoadWorkspaceContext() error = %v", err)
	}
	if ctx.WorkspaceDir != wsDir {
		t.Fatalf("WorkspaceDir = %q, want %q", ctx.WorkspaceDir, wsDir)
	}
	if ctx.AGENTSMD != "" {
		t.Fatalf("AGENTSMD = %q, want empty string", ctx.AGENTSMD)
	}
}

func TestKanbanWorkspaceAGENTSLoadRejectsEmptyPath(t *testing.T) {
	_, err := LoadWorkspaceContext("")
	if err == nil {
		t.Fatal("LoadWorkspaceContext('') = nil, want error")
	}
}

func TestKanbanWorkspaceAGENTSLoadRejectsFilePath(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "not_a_dir")
	if err := os.WriteFile(filePath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := LoadWorkspaceContext(filePath)
	if err == nil {
		t.Fatal("LoadWorkspaceContext(file) = nil, want error for non-directory path")
	}
}

func TestKanbanWorkspaceContextInjectionWorktree(t *testing.T) {
	wsDir := t.TempDir()
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { store.Close() })

	task, err := store.CreateTask(ctx, CreateTaskInput{
		Title:         "worktree workspace task",
		Assignee:      "worker-c",
		WorkspaceKind: WorkspaceWorktree,
		WorkspacePath: wsDir,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	resolved, err := store.resolveWorkspace(ctx, task)
	if err != nil {
		t.Fatalf("resolveWorkspace() error = %v", err)
	}
	if resolved != wsDir {
		t.Fatalf("resolveWorkspace() = %q, want %q", resolved, wsDir)
	}
}

func TestKanbanWorkspaceContextInjectionMissingDirRejected(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "kanban.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { store.Close() })

	task, err := store.CreateTask(ctx, CreateTaskInput{
		Title:         "bad workspace task",
		Assignee:      "worker-d",
		WorkspaceKind: WorkspaceDir,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	_, err = store.resolveWorkspace(ctx, task)
	if err == nil {
		t.Fatal("resolveWorkspace() with empty path for WorkspaceDir = nil, want error")
	}
}

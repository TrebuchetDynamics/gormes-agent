package kanban

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestKanbanProcessSpawnerBuildsNativeWorkerCommandAndRotatesLogs(t *testing.T) {
	ctx := context.Background()
	workspace := t.TempDir()
	logRoot := filepath.Join(t.TempDir(), "logs")
	if err := os.MkdirAll(logRoot, 0o755); err != nil {
		t.Fatalf("mkdir log root: %v", err)
	}
	logPath := filepath.Join(logRoot, "t_worker.log")
	if err := os.WriteFile(logPath, []byte("old oversized log"), 0o600); err != nil {
		t.Fatalf("write old log: %v", err)
	}

	now := time.Date(2026, 5, 6, 18, 0, 0, 0, time.UTC)
	starter := &recordingProcessStarter{
		result: ProcessStartResult{PID: 4242, StartedAt: now},
	}
	spawner := ProcessSpawner{
		Binary:      "/opt/gormes/bin/gormes",
		LogRoot:     logRoot,
		MaxLogBytes: 4,
		Starter:     starter,
		Now:         func() time.Time { return now },
		BinaryLookPath: func(string) (string, error) {
			t.Fatal("BinaryLookPath should not be called for explicit Binary")
			return "", nil
		},
		ExecutablePath: func() (string, error) {
			t.Fatal("ExecutablePath should not be called for explicit Binary")
			return "", nil
		},
	}

	result, err := spawner.SpawnKanbanWorker(ctx, SpawnRequest{
		Task: Task{
			ID:       "t_worker",
			Title:    "Ship worker binding",
			Assignee: "coder",
		},
		WorkspacePath: workspace,
		Env: map[string]string{
			"GORMES_KANBAN_DB":        filepath.Join(t.TempDir(), "kanban.db"),
			"GORMES_KANBAN_WORKSPACE": workspace,
			"GORMES_PROFILE":          "coder",
			"HERMES_HOME":             "/tmp/must-not-leak",
			"HERMES_KANBAN_DB":        "/tmp/must-not-leak.db",
		},
	})
	if err != nil {
		t.Fatalf("SpawnKanbanWorker() error = %v", err)
	}
	if result.PID != 4242 {
		t.Fatalf("PID = %d, want 4242", result.PID)
	}
	if len(starter.requests) != 1 {
		t.Fatalf("starter requests = %d, want 1", len(starter.requests))
	}
	req := starter.requests[0]
	if req.Binary != "/opt/gormes/bin/gormes" {
		t.Fatalf("Binary = %q", req.Binary)
	}
	wantArgs := []string{"-p", "coder", "--skills", "kanban-worker", "chat", "-q", "work kanban task t_worker"}
	if !reflect.DeepEqual(req.Args, wantArgs) {
		t.Fatalf("Args = %#v, want %#v", req.Args, wantArgs)
	}
	if req.Dir != workspace {
		t.Fatalf("Dir = %q, want workspace %q", req.Dir, workspace)
	}
	for _, key := range []string{"GORMES_KANBAN_DB", "GORMES_KANBAN_TASK", "GORMES_KANBAN_WORKSPACE", "GORMES_PROFILE"} {
		if req.Env[key] == "" {
			t.Fatalf("Env[%s] is empty in %#v", key, req.Env)
		}
	}
	for key := range req.Env {
		if len(key) >= len("HERMES") && key[:len("HERMES")] == "HERMES" {
			t.Fatalf("starter env leaked Hermes key %s in %#v", key, req.Env)
		}
	}
	if req.StdoutPath != logPath || req.StderrPath != logPath {
		t.Fatalf("stdout/stderr paths = %q/%q, want %q", req.StdoutPath, req.StderrPath, logPath)
	}
	if _, err := os.Stat(logPath + ".1"); err != nil {
		t.Fatalf("rotated log missing: %v", err)
	}
	if info, err := os.Stat(logPath); err != nil {
		t.Fatalf("new log missing: %v", err)
	} else if info.Size() != 0 {
		t.Fatalf("new log size = %d, want empty", info.Size())
	}
}

func TestKanbanProcessSpawnerNamedBoardLogRoot(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	workspace := t.TempDir()
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	starter := &recordingProcessStarter{
		result: ProcessStartResult{PID: 5150, StartedAt: now},
	}
	spawner := ProcessSpawner{
		Starter: starter,
		Now:     func() time.Time { return now },
	}

	_, err := spawner.SpawnKanbanWorker(ctx, SpawnRequest{
		Task: Task{
			ID:       "t_named",
			Title:    "Named board log root",
			Assignee: "coder",
		},
		WorkspacePath: workspace,
		Env: map[string]string{
			"GORMES_KANBAN_DB": filepath.Join(root, "kanban", "boards", "alpha", "kanban.db"),
		},
	})
	if err != nil {
		t.Fatalf("SpawnKanbanWorker() error = %v", err)
	}
	if len(starter.requests) != 1 {
		t.Fatalf("starter requests = %d, want 1", len(starter.requests))
	}
	want := filepath.Join(root, "kanban", "boards", "alpha", "logs", "t_named.log")
	req := starter.requests[0]
	if req.StdoutPath != want || req.StderrPath != want {
		t.Fatalf("stdout/stderr paths = %q/%q, want %q", req.StdoutPath, req.StderrPath, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("log path missing: %v", err)
	}
}

func TestKanbanProcessSpawnerDefaultBinaryPrefersPathShim(t *testing.T) {
	ctx := context.Background()
	workspace := t.TempDir()
	now := time.Date(2026, 5, 10, 18, 0, 0, 0, time.UTC)
	starter := &recordingProcessStarter{
		result: ProcessStartResult{PID: 6161, StartedAt: now},
	}
	spawner := ProcessSpawner{
		LogRoot: filepath.Join(t.TempDir(), "logs"),
		Starter: starter,
		Now:     func() time.Time { return now },
		BinaryLookPath: func(name string) (string, error) {
			if name != "gormes" {
				t.Fatalf("BinaryLookPath name = %q, want gormes", name)
			}
			return "/usr/local/bin/gormes", nil
		},
		ExecutablePath: func() (string, error) {
			t.Fatal("ExecutablePath should not be called when gormes is on PATH")
			return "", nil
		},
	}

	_, err := spawner.SpawnKanbanWorker(ctx, SpawnRequest{
		Task: Task{
			ID:       "t_path",
			Title:    "PATH shim wins",
			Assignee: "coder",
		},
		WorkspacePath: workspace,
		Env: map[string]string{
			"GORMES_KANBAN_DB": filepath.Join(t.TempDir(), "kanban.db"),
		},
	})
	if err != nil {
		t.Fatalf("SpawnKanbanWorker() error = %v", err)
	}
	if len(starter.requests) != 1 {
		t.Fatalf("starter requests = %d, want 1", len(starter.requests))
	}
	if got := starter.requests[0].Binary; got != "/usr/local/bin/gormes" {
		t.Fatalf("Binary = %q, want PATH shim", got)
	}
}

func TestKanbanProcessSpawnerDefaultBinaryFallsBackToCurrentExecutable(t *testing.T) {
	ctx := context.Background()
	workspace := t.TempDir()
	now := time.Date(2026, 5, 10, 18, 30, 0, 0, time.UTC)
	starter := &recordingProcessStarter{
		result: ProcessStartResult{PID: 7171, StartedAt: now},
	}
	spawner := ProcessSpawner{
		LogRoot: filepath.Join(t.TempDir(), "logs"),
		Starter: starter,
		Now:     func() time.Time { return now },
		BinaryLookPath: func(name string) (string, error) {
			if name != "gormes" {
				t.Fatalf("BinaryLookPath name = %q, want gormes", name)
			}
			return "", os.ErrNotExist
		},
		ExecutablePath: func() (string, error) {
			return "/opt/gormes/current/gormes", nil
		},
	}

	_, err := spawner.SpawnKanbanWorker(ctx, SpawnRequest{
		Task: Task{
			ID:       "t_fallback",
			Title:    "Current executable fallback",
			Assignee: "coder",
		},
		WorkspacePath: workspace,
		Env: map[string]string{
			"GORMES_KANBAN_DB": filepath.Join(t.TempDir(), "kanban.db"),
		},
	})
	if err != nil {
		t.Fatalf("SpawnKanbanWorker() error = %v", err)
	}
	if len(starter.requests) != 1 {
		t.Fatalf("starter requests = %d, want 1", len(starter.requests))
	}
	if got := starter.requests[0].Binary; got != "/opt/gormes/current/gormes" {
		t.Fatalf("Binary = %q, want current executable fallback", got)
	}
}

type recordingProcessStarter struct {
	requests []ProcessStartRequest
	result   ProcessStartResult
	err      error
}

func (s *recordingProcessStarter) StartKanbanProcess(_ context.Context, req ProcessStartRequest) (ProcessStartResult, error) {
	s.requests = append(s.requests, req)
	return s.result, s.err
}

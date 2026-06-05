package gormescli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKanbanLogCommandPrintsFullAndTail(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GORMES_HOME", root)

	task := runKanbanJSONTask(t, "create", "Loggable task", "--json")
	logRoot := filepath.Join(root, "kanban", "logs")
	if err := os.MkdirAll(logRoot, 0o755); err != nil {
		t.Fatalf("create log root: %v", err)
	}
	logPath := filepath.Join(logRoot, task.ID+".log")
	if err := os.WriteFile(logPath, []byte("line 0\nline 1\nline 2\nline 3"), 0o644); err != nil {
		t.Fatalf("write worker log fixture: %v", err)
	}

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "log", task.ID)
	if err != nil {
		t.Fatalf("kanban log: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("kanban log stderr = %q, want empty", stderr)
	}
	if stdout != "line 0\nline 1\nline 2\nline 3\n" {
		t.Fatalf("kanban log stdout = %q, want full log with appended newline", stdout)
	}

	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "log", task.ID, "--tail", "16")
	if err != nil {
		t.Fatalf("kanban log --tail: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if stdout != "line 2\nline 3\n" {
		t.Fatalf("kanban log --tail stdout = %q, want bounded tail", stdout)
	}
}

func TestKanbanLogCommandMissingLogFailsBounded(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GORMES_HOME", root)

	task := runKanbanJSONTask(t, "create", "No log yet", "--json")
	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "log", task.ID)
	if err == nil {
		t.Fatalf("kanban log missing succeeded\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	combined := stdout + stderr + err.Error()
	if !strings.Contains(combined, "no log for "+task.ID) {
		t.Fatalf("missing log output = %q, want bounded no-log evidence", combined)
	}
	if strings.Contains(combined, root+"/kanban/logs") {
		t.Fatalf("missing log output leaked local log root: %q", combined)
	}
	if _, statErr := os.Stat(filepath.Join(root, "kanban", "logs", task.ID+".log")); !os.IsNotExist(statErr) {
		t.Fatalf("missing log command created log file or stat failed: %v", statErr)
	}
}

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kanban"
)

func TestKanbanCommandUsesGormesHomeNotHermesHome(t *testing.T) {
	root := t.TempDir()
	gormesHome := filepath.Join(root, "gormes")
	hermesHome := filepath.Join(root, "hermes")
	t.Setenv("GORMES_HOME", gormesHome)
	t.Setenv("HERMES_HOME", hermesHome)

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "init")
	if err != nil {
		t.Fatalf("kanban init error = %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, filepath.Join(gormesHome, "kanban.db")) {
		t.Fatalf("kanban init stdout = %q, want GORMES_HOME kanban.db", stdout)
	}
	if _, err := os.Stat(filepath.Join(gormesHome, "kanban.db")); err != nil {
		t.Fatalf("Gormes kanban.db not created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(hermesHome, "kanban.db")); !os.IsNotExist(err) {
		t.Fatalf("Hermes kanban.db touched, stat err = %v", err)
	}
}

func TestKanbanCommandCreateWithParentPromotesAfterComplete(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())

	parent := runKanbanJSONTask(t, "create", "Design schema", "--assignee", "researcher", "--json")
	if parent.Status != kanban.StatusReady || parent.Assignee != "researcher" {
		t.Fatalf("parent = %+v, want ready researcher", parent)
	}

	child := runKanbanJSONTask(t, "create", "Implement API", "--assignee", "backend-dev", "--parent", parent.ID, "--json")
	if child.Status != kanban.StatusTodo {
		t.Fatalf("child status = %q, want %q", child.Status, kanban.StatusTodo)
	}

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "complete", parent.ID, "--result", "schema ready")
	if err != nil {
		t.Fatalf("kanban complete error = %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}

	got := runKanbanJSONTask(t, "show", child.ID, "--json")
	if got.Status != kanban.StatusReady {
		t.Fatalf("child after complete status = %q, want %q", got.Status, kanban.StatusReady)
	}
}

func TestKanbanCommandListJSONFiltersByStatus(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())

	ready := runKanbanJSONTask(t, "create", "Ready task", "--json")
	parent := runKanbanJSONTask(t, "create", "Parent", "--json")
	_ = runKanbanJSONTask(t, "create", "Todo child", "--parent", parent.ID, "--json")

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "kanban", "list", "--status", string(kanban.StatusReady), "--json")
	if err != nil {
		t.Fatalf("kanban list --json error = %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var tasks []kanban.Task
	if err := json.Unmarshal([]byte(stdout), &tasks); err != nil {
		t.Fatalf("list JSON decode error = %v\nstdout=%s", err, stdout)
	}
	var titles []string
	for _, task := range tasks {
		if task.Status != kanban.StatusReady {
			t.Fatalf("filtered task status = %q, want only ready: %+v", task.Status, tasks)
		}
		titles = append(titles, task.Title)
	}
	if !containsKanbanString(titles, ready.Title) || !containsKanbanString(titles, parent.Title) || containsKanbanString(titles, "Todo child") {
		t.Fatalf("ready titles = %v, want ready+parent and no todo child", titles)
	}
}

func runKanbanJSONTask(t *testing.T, args ...string) kanban.Task {
	t.Helper()
	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), append([]string{"kanban"}, args...)...)
	if err != nil {
		t.Fatalf("gormes kanban %v error = %v\nstdout=%s\nstderr=%s", args, err, stdout, stderr)
	}
	var task kanban.Task
	if err := json.Unmarshal([]byte(stdout), &task); err != nil {
		t.Fatalf("gormes kanban %v JSON decode error = %v\nstdout=%s", args, err, stdout)
	}
	return task
}

func containsKanbanString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

package tools

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckpointManagerRoundTripDiffAndRestore(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	project, filePath := newCheckpointTestProject(t, "v1\n")
	mgr := NewCheckpointManager(CheckpointManagerConfig{
		Enabled:      true,
		MaxSnapshots: 10,
	})

	if ok := mgr.EnsureCheckpoint(project, "initial"); !ok {
		t.Fatal("EnsureCheckpoint(initial) = false, want true")
	}
	if _, err := os.Stat(filepath.Join(project, ".git")); !os.IsNotExist(err) {
		t.Fatalf("project .git stat err = %v, want not exists", err)
	}

	checkpoints, err := mgr.List(project)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(checkpoints) != 1 {
		t.Fatalf("List() len = %d, want 1", len(checkpoints))
	}
	initial := checkpoints[0]

	if err := os.WriteFile(filePath, []byte("v2\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(v2): %v", err)
	}
	diff, err := mgr.Diff(project, initial.Hash)
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}
	if !strings.Contains(diff.Diff, "-v1") || !strings.Contains(diff.Diff, "+v2") {
		t.Fatalf("Diff() output = %q, want v1->v2 delta", diff.Diff)
	}

	restore, err := mgr.Restore(project, initial.Hash, "")
	if err != nil {
		t.Fatalf("Restore() error = %v", err)
	}
	got, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile(restored): %v", err)
	}
	if string(got) != "v1\n" {
		t.Fatalf("restored file = %q, want %q", string(got), "v1\n")
	}
	if restore.RestoredTo != initial.ShortHash {
		t.Fatalf("Restore().RestoredTo = %q, want %q", restore.RestoredTo, initial.ShortHash)
	}

	checkpoints, err = mgr.List(project)
	if err != nil {
		t.Fatalf("List() after restore error = %v", err)
	}
	if len(checkpoints) != 2 {
		t.Fatalf("List() after restore len = %d, want 2", len(checkpoints))
	}
	if !strings.Contains(checkpoints[0].Reason, "pre-rollback snapshot") {
		t.Fatalf("latest checkpoint reason = %q, want pre-rollback snapshot", checkpoints[0].Reason)
	}
}

func TestCheckpointManagerDeduplicatesPerTurn(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	project, filePath := newCheckpointTestProject(t, "first\n")
	mgr := NewCheckpointManager(CheckpointManagerConfig{
		Enabled:      true,
		MaxSnapshots: 10,
	})

	if ok := mgr.EnsureCheckpoint(project, "turn one"); !ok {
		t.Fatal("EnsureCheckpoint(turn one) = false, want true")
	}
	if err := os.WriteFile(filePath, []byte("second\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(second): %v", err)
	}
	if ok := mgr.EnsureCheckpoint(project, "same turn"); ok {
		t.Fatal("EnsureCheckpoint(same turn) = true, want false")
	}

	mgr.NewTurn()
	if ok := mgr.EnsureCheckpoint(project, "turn two"); !ok {
		t.Fatal("EnsureCheckpoint(turn two) = false, want true")
	}

	checkpoints, err := mgr.List(project)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(checkpoints) != 2 {
		t.Fatalf("List() len = %d, want 2", len(checkpoints))
	}
	if checkpoints[0].Reason != "turn two" || checkpoints[1].Reason != "turn one" {
		t.Fatalf("checkpoint reasons = [%q %q], want [turn two turn one]", checkpoints[0].Reason, checkpoints[1].Reason)
	}
}

func newCheckpointTestProject(t *testing.T, content string) (project string, filePath string) {
	t.Helper()

	root := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "xdg-data"))

	project = filepath.Join(root, "project")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatalf("MkdirAll(project): %v", err)
	}
	filePath = filepath.Join(project, "main.txt")
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(initial): %v", err)
	}
	return project, filePath
}

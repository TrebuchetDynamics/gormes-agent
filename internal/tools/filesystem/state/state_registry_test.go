package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileStateRegistryDetectsStaleWrites(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(path, []byte("alpha\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	registry := NewFileStateRegistry()
	snapshot, err := registry.Record(root, "agent-a", root, "notes.txt", path)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if snapshot.TaskID != "agent-a" || snapshot.Path != "notes.txt" || snapshot.ReadToken == "" {
		t.Fatalf("snapshot lacks stable task/path/token evidence: %+v", snapshot)
	}
	if check := registry.Check(root, "agent-a", root, "notes.txt", path); check != nil {
		t.Fatalf("Check before mutation = %+v, want nil", check)
	}

	if err := os.WriteFile(path, []byte("external\n"), 0o644); err != nil {
		t.Fatalf("external write: %v", err)
	}
	check := registry.Check(root, "agent-a", root, "notes.txt", path)
	if check == nil {
		t.Fatalf("Check after mutation = nil, want stale evidence")
	}
	if check.Status != FileStateStatusStale || check.Expected.ReadToken != snapshot.ReadToken || check.Current == nil {
		t.Fatalf("stale check = %+v, want stale expected/current evidence", check)
	}
}

func TestNormalizeFileTaskIDDefaultsBlankTasks(t *testing.T) {
	if got := NormalizeFileTaskID(" \t"); got != "default" {
		t.Fatalf("NormalizeFileTaskID(blank) = %q, want default", got)
	}
}

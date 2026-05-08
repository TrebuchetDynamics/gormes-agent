package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func seedCheckpointShadow(t *testing.T, root, name, workdir string, mtime time.Time) string {
	t.Helper()
	shadow := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Join(shadow, "refs", "heads"), 0o755); err != nil {
		t.Fatalf("mkdir shadow %s: %v", shadow, err)
	}
	if err := os.WriteFile(filepath.Join(shadow, "HEAD"), []byte("ref: refs/heads/main\n"), 0o644); err != nil {
		t.Fatalf("write HEAD: %v", err)
	}
	if err := os.WriteFile(filepath.Join(shadow, "GORMES_WORKDIR"), []byte(workdir+"\n"), 0o644); err != nil {
		t.Fatalf("write GORMES_WORKDIR: %v", err)
	}
	if err := os.WriteFile(filepath.Join(shadow, "refs", "heads", "main"), []byte("abc123\n"), 0o644); err != nil {
		t.Fatalf("write ref: %v", err)
	}
	// Set modtimes on shadow contents so newestCheckpointMTime finds the injected time.
	_ = os.Chtimes(filepath.Join(shadow, "HEAD"), mtime, mtime)
	_ = os.Chtimes(filepath.Join(shadow, "GORMES_WORKDIR"), mtime, mtime)
	_ = os.Chtimes(filepath.Join(shadow, "refs", "heads", "main"), mtime, mtime)
	_ = os.Chtimes(filepath.Join(shadow, "refs", "heads"), mtime, mtime)
	_ = os.Chtimes(filepath.Join(shadow, "refs"), mtime, mtime)
	if err := os.Chtimes(shadow, mtime, mtime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	return shadow
}

func seedLegacyArchive(t *testing.T, root, name string) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir legacy %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "data.txt"), []byte("legacy content\n"), 0o644); err != nil {
		t.Fatalf("write legacy data: %v", err)
	}
}

func checkpointTestRoot(tmp string) string {
	return filepath.Join(tmp, "gormes", "checkpoints")
}

func TestCheckpointsCLI_Status_ReportsStoreDetails(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	root := checkpointTestRoot(tmp)
	os.MkdirAll(root, 0o755)

	now := time.Date(2026, 5, 8, 10, 0, 0, 0, time.UTC)
	workdir := filepath.Join(tmp, "project-a")
	os.MkdirAll(workdir, 0o755)
	seedCheckpointShadow(t, root, "shadow-a", workdir, now.Add(-30*time.Minute))

	cmd := newCheckpointsCommand()
	cmd.SetArgs([]string{"status"})
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("checkpoints status: %v", err)
	}

	output := out.String()
	for _, want := range []string{
		"Checkpoint base:",
		"Total size:",
		"store/",
		"Projects:",
		"project-a",
		"live",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q\n%s", want, output)
		}
	}
}

func TestCheckpointsCLI_List_IsStatusAlias(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	root := checkpointTestRoot(tmp)
	os.MkdirAll(root, 0o755)

	now := time.Date(2026, 5, 8, 10, 0, 0, 0, time.UTC)
	workdir := filepath.Join(tmp, "proj")
	os.MkdirAll(workdir, 0o755)
	seedCheckpointShadow(t, root, "s1", workdir, now)

	statusOut := &bytes.Buffer{}
	listOut := &bytes.Buffer{}

	cmd := newCheckpointsCommand()
	cmd.SetArgs([]string{"status"})
	cmd.SetOut(statusOut)
	_ = cmd.Execute()

	cmd = newCheckpointsCommand()
	cmd.SetArgs([]string{"list"})
	cmd.SetOut(listOut)
	_ = cmd.Execute()

	if statusOut.String() != listOut.String() {
		t.Errorf("list output differs from status\nstatus:\n%s\nlist:\n%s",
			statusOut.String(), listOut.String())
	}
}

func TestCheckpointsCLI_Prune_DeletesOrphanAndStale(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	root := checkpointTestRoot(tmp)
	os.MkdirAll(root, 0o755)

	now := time.Date(2026, 5, 8, 10, 0, 0, 0, time.UTC)
	activeWorkdir := filepath.Join(tmp, "active")
	os.MkdirAll(activeWorkdir, 0o755)
	seedCheckpointShadow(t, root, "active-shadow", activeWorkdir, now.Add(-1*time.Hour))

	orphanWorkdir := filepath.Join(tmp, "deleted-project")
	seedCheckpointShadow(t, root, "orphan-shadow", orphanWorkdir, now.Add(-2*time.Hour))

	staleWorkdir := filepath.Join(tmp, "stale")
	os.MkdirAll(staleWorkdir, 0o755)
	seedCheckpointShadow(t, root, "stale-shadow", staleWorkdir, now.Add(-10*24*time.Hour))

	t.Setenv("XDG_DATA_HOME", tmp)

	cmd := newCheckpointsCommand()
	cmd.SetArgs([]string{"prune", "--retention-days", "5", "--max-size-mb", "500"})
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("checkpoints prune: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Deleted orphan:  1") {
		t.Errorf("expected 1 deleted orphan\n%s", output)
	}
	if !strings.Contains(output, "Deleted stale:   1") {
		t.Errorf("expected 1 deleted stale\n%s", output)
	}

	if _, err := os.Stat(filepath.Join(root, "active-shadow")); err != nil {
		t.Errorf("active shadow was pruned: %v", err)
	}
}

func TestCheckpointsCLI_Prune_KeepOrphansFlag(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	root := checkpointTestRoot(tmp)
	os.MkdirAll(root, 0o755)

	now := time.Date(2026, 5, 8, 10, 0, 0, 0, time.UTC)
	orphanWorkdir := filepath.Join(tmp, "deleted")
	seedCheckpointShadow(t, root, "orphan", orphanWorkdir, now)

	t.Setenv("XDG_DATA_HOME", tmp)

	cmd := newCheckpointsCommand()
	cmd.SetArgs([]string{"prune", "--keep-orphans", "--retention-days", "1"})
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("checkpoints prune --keep-orphans: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Deleted orphan:  0") {
		t.Errorf("expected 0 deleted orphan with --keep-orphans\n%s", output)
	}
	if _, err := os.Stat(filepath.Join(root, "orphan")); err != nil {
		t.Errorf("orphan was deleted despite --keep-orphans: %v", err)
	}
}

func TestCheckpointsCLI_Prune_DefaultRetentionAndSize(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "checkpoints")
	os.MkdirAll(root, 0o755)

	t.Setenv("XDG_DATA_HOME", tmp)

	cmd := newCheckpointsCommand()
	cmd.SetArgs([]string{"prune"})
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("checkpoints prune (defaults): %v", err)
	}

	output := out.String()
	for _, want := range []string{
		"retention_days:    7",
		"max_total_size_mb: 500",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q\n%s", want, output)
		}
	}
}

func TestCheckpointsCLI_Clear_WithForceFlag(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	root := checkpointTestRoot(tmp)
	os.MkdirAll(root, 0o755)
	os.WriteFile(filepath.Join(root, "placeholder"), []byte("x"), 0o644)

	t.Setenv("XDG_DATA_HOME", tmp)

	cmd := newCheckpointsCommand()
	cmd.SetArgs([]string{"clear", "-f"})
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("checkpoints clear -f: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Cleared.") {
		t.Errorf("expected Cleared.\n%s", output)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Errorf("root still exists after clear -f: %v", err)
	}
}

func TestCheckpointsCLI_ClearLegacy_ReportsArchives(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	root := checkpointTestRoot(tmp)
	os.MkdirAll(root, 0o755)

	seedLegacyArchive(t, root, "legacy-1")
	seedLegacyArchive(t, root, "legacy-2")

	t.Setenv("XDG_DATA_HOME", tmp)

	cmd := newCheckpointsCommand()
	cmd.SetArgs([]string{"clear-legacy", "-f"})
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("checkpoints clear-legacy -f: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Found 2 legacy archive") {
		t.Errorf("expected 2 legacy archives reported\n%s", output)
	}
	if _, err := os.Stat(filepath.Join(root, "legacy-1")); !os.IsNotExist(err) {
		t.Errorf("legacy-1 still exists: %v", err)
	}
}

func TestCheckpointsCLI_EmptyStoreNoPanic(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	cmd := newCheckpointsCommand()
	cmd.SetArgs([]string{"status"})
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("checkpoints status on empty store: %v", err)
	}

	if !strings.Contains(out.String(), "No checkpoint store") {
		t.Logf("output: %s", out.String())
	}
}

func TestCheckpointsCLI_BareCheckpointsDefaultsToStatus(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	root := checkpointTestRoot(tmp)
	os.MkdirAll(root, 0o755)

	t.Setenv("XDG_DATA_HOME", tmp)

	bareOut := &bytes.Buffer{}
	statusOut := &bytes.Buffer{}

	cmd := newCheckpointsCommand()
	cmd.SetOut(bareOut)
	_ = cmd.Execute()

	cmd = newCheckpointsCommand()
	cmd.SetArgs([]string{"status"})
	cmd.SetOut(statusOut)
	_ = cmd.Execute()

	bare := bareOut.String()
	status := statusOut.String()
	if bare != status {
		t.Errorf("bare checkpoints output differs from status\nbare:\n%s\nstatus:\n%s", bare, status)
	}
}

func TestCheckpointsCLI_NoProviderSubmits(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "checkpoints")
	os.MkdirAll(root, 0o755)

	t.Setenv("XDG_DATA_HOME", tmp)

	for _, args := range [][]string{
		{"status"},
		{"list"},
		{"prune"},
		{"clear", "-f"},
		{"clear-legacy", "-f"},
	} {
		cmd := newCheckpointsCommand()
		cmd.SetArgs(args)
		cmd.SetOut(&bytes.Buffer{})
		if err := cmd.Execute(); err != nil {
			t.Errorf("checkpoints %s: %v", strings.Join(args, " "), err)
		}
	}

	_ = tools.DefaultCheckpointRoot()
}

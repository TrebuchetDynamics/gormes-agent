package gormescli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Live regression 2026-05-10: a sandbox uninstall test accidentally
// targeted the operator's REAL ~/.gormes (because install.sh's
// run_uninstall did not export GORMES_HOME), and the uninstall PERMANENTLY
// DELETED .env (provider keys), memory.db (Goncho conversation history),
// config.toml, and the binary. Recovery worked only because an earlier
// uninstall on May 2 had moved files to the freedesktop trash. These
// tests pin two parts of today's defense-in-depth fix:
//
//   1. The default removal mode prefers `gio trash` when available so
//      uninstalled artifacts stay recoverable from the file manager's
//      trash; permanent delete falls through only when gio is absent.
//   2. Operators can opt into permanent delete via
//      GORMES_UNINSTALL_FORCE_PURGE=1 for environments that legitimately
//      need scrubbed disk (CI cleanup, container teardown, secure wipe).

func TestPickArtifactMover_PrefersGioTrashWhenAvailable(t *testing.T) {
	if _, err := exec.LookPath("gio"); err != nil {
		t.Skip("gio not on PATH; can't exercise the trash-aware default path on this host")
	}
	t.Setenv("GORMES_UNINSTALL_FORCE_PURGE", "")

	mover := pickArtifactMover()
	if !strings.Contains(strings.ToLower(mover.label), "trash") {
		t.Fatalf("expected trash-aware mover label when gio is available, got %q", mover.label)
	}
	if !strings.Contains(strings.ToLower(mover.label), "recoverable") {
		t.Fatalf("mover label should advertise recoverable trash to operators, got %q", mover.label)
	}
}

func TestPickArtifactMover_ForcePurgeOptsIntoPermanentDelete(t *testing.T) {
	t.Setenv("GORMES_UNINSTALL_FORCE_PURGE", "1")

	mover := pickArtifactMover()
	if !strings.Contains(strings.ToLower(mover.label), "permanent delete") {
		t.Fatalf("expected permanent-delete mover when GORMES_UNINSTALL_FORCE_PURGE=1, got %q", mover.label)
	}

	// And it must actually delete on disk, not move to trash.
	tmp := t.TempDir()
	doomed := filepath.Join(tmp, "doomed.txt")
	if err := os.WriteFile(doomed, []byte("bye"), 0o644); err != nil {
		t.Fatalf("seed doomed file: %v", err)
	}
	if err := mover.move(doomed); err != nil {
		t.Fatalf("mover.move returned error under force-purge: %v", err)
	}
	if _, err := os.Stat(doomed); !os.IsNotExist(err) {
		t.Fatalf("force-purge should have removed %s; stat err=%v", doomed, err)
	}
}

func TestPickArtifactMover_ForcePurgeAcceptsTrueAlias(t *testing.T) {
	t.Setenv("GORMES_UNINSTALL_FORCE_PURGE", "true")

	mover := pickArtifactMover()
	if !strings.Contains(strings.ToLower(mover.label), "permanent delete") {
		t.Fatalf("expected permanent-delete mover when GORMES_UNINSTALL_FORCE_PURGE=true, got %q", mover.label)
	}
}

func TestPickArtifactMover_DefaultOnHostWithoutGio(t *testing.T) {
	// Simulate "no gio" by stubbing PATH to a directory that contains no
	// commands. We don't actually move a file in the gio-available branch
	// because that would litter the real Trash dir; we only assert the
	// label contract.
	if _, err := exec.LookPath("gio"); err == nil {
		// Override PATH so LookPath("gio") fails for this subtest.
		t.Setenv("PATH", t.TempDir())
	}
	t.Setenv("GORMES_UNINSTALL_FORCE_PURGE", "")

	mover := pickArtifactMover()
	if !strings.Contains(strings.ToLower(mover.label), "permanent delete") {
		t.Fatalf("expected permanent-delete fallback when gio is absent, got %q", mover.label)
	}
	if !strings.Contains(strings.ToLower(mover.label), "gio not available") {
		t.Fatalf("fallback label should explain why permanent delete is in use, got %q", mover.label)
	}
}

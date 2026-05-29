package cli

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestWriteBackupZip_IncludesPlainFilesExcludesSidecars proves the
// writer walks sourceDir, picks up plain files, and skips paths that
// match IsExcludedFromBackup (checkpoints/, *.db-wal, *.db-shm,
// *.db-journal). The resulting zip's entry list is the contract.
func TestWriteBackupZip_IncludesPlainFilesExcludesSidecars(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "pre-update.zip")

	// Layout the writer must observe.
	mustWrite := func(rel, body string) {
		full := filepath.Join(src, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	mustWrite("config.toml", "_config_version = 19\n")
	mustWrite("auth.json", `{"credential_pool":{}}`)
	mustWrite("checkpoints/snap.bin", "should be skipped")
	mustWrite("sessions/log.db", "real db")
	mustWrite("sessions/log.db-wal", "transient sidecar")
	mustWrite("sessions/log.db-shm", "transient sidecar")
	mustWrite("sessions/log.db-journal", "transient sidecar")
	mustWrite("nested/dir/skill.md", "nested ok")

	res, err := WriteBackupZip(context.Background(), src, dst)
	if err != nil {
		t.Fatalf("WriteBackupZip: %v", err)
	}
	if res.Path != dst {
		t.Fatalf("BackupResult.Path = %q, want %q", res.Path, dst)
	}
	if res.SizeBytes <= 0 {
		t.Fatalf("BackupResult.SizeBytes must be > 0; got %d", res.SizeBytes)
	}
	if res.DurationMs < 0 {
		t.Fatalf("BackupResult.DurationMs must be >= 0; got %d", res.DurationMs)
	}

	got := zipEntries(t, dst)
	sort.Strings(got)
	want := []string{
		"auth.json",
		"config.toml",
		"nested/dir/skill.md",
		"sessions/log.db",
	}
	if len(got) != len(want) {
		t.Fatalf("entries = %v, want %v", got, want)
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("entry[%d] = %q, want %q (full: %v)", i, got[i], w, got)
		}
	}
}

// TestWriteBackupZip_AtomicRenameOnSuccessNoTmpLeak proves the writer's
// atomic-rename invariant: after a successful write, no .tmp file
// remains alongside the destination zip.
func TestWriteBackupZip_AtomicRenameOnSuccessNoTmpLeak(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "x.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	dst := filepath.Join(t.TempDir(), "out.zip")
	if _, err := WriteBackupZip(context.Background(), src, dst); err != nil {
		t.Fatalf("WriteBackupZip: %v", err)
	}
	if _, err := os.Stat(dst + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf(".tmp file must not remain after a successful rename; stat err = %v", err)
	}
}

// TestWriteBackupZip_RejectsMissingSourceDir proves the writer fails
// fast with an operator-friendly "source dir not found" message when
// the caller hands it a path that doesn't exist. Without this guard,
// the underlying filepath.WalkDir error chain leaks "lstat /path: no
// such file or directory" through a "backup: walk/write:" wrapper —
// confusing operators with a syscall name and wording that suggests
// mid-write failure rather than a missing input.
func TestWriteBackupZip_RejectsMissingSourceDir(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-source-dir")
	dst := filepath.Join(t.TempDir(), "out.zip")
	_, err := WriteBackupZip(context.Background(), missing, dst)
	if err == nil {
		t.Fatal("missing source dir must error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "source dir not found") {
		t.Fatalf("err must say `source dir not found`; got %q", msg)
	}
	if !strings.Contains(msg, missing) {
		t.Fatalf("err must include the offending path; got %q", msg)
	}
	if strings.Contains(msg, "lstat") || strings.Contains(msg, "walk/write") {
		t.Fatalf("err must NOT leak syscall name or walk-wording; got %q", msg)
	}
	// Atomic-rename invariant must still hold: no partial .tmp left
	// behind on a fast-fail.
	if _, statErr := os.Stat(dst + ".tmp"); !os.IsNotExist(statErr) {
		t.Fatalf("fast-fail must not leave a tmp file; got statErr=%v", statErr)
	}
}

// TestWriteBackupZip_RejectsEmptyArgs proves the writer guards its inputs
// so a misconfigured caller fails fast instead of producing a corrupt or
// surprising artifact.
func TestWriteBackupZip_RejectsEmptyArgs(t *testing.T) {
	if _, err := WriteBackupZip(context.Background(), "", "/tmp/x.zip"); err == nil {
		t.Fatalf("empty sourceDir must be rejected")
	}
	if _, err := WriteBackupZip(context.Background(), "/tmp", ""); err == nil {
		t.Fatalf("empty destPath must be rejected")
	}
}

// zipEntries reads zipPath and returns the slash-separated file names it
// contains, in directory-order.
func zipEntries(t *testing.T, zipPath string) []string {
	t.Helper()
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("zip.OpenReader: %v", err)
	}
	defer r.Close()
	out := make([]string, 0, len(r.File))
	for _, f := range r.File {
		out = append(out, f.Name)
	}
	return out
}

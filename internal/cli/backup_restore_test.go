package cli

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRestoreFromZip_ExtractsAllEntries proves the rollback path: a
// pre-update-*.zip created by WriteBackupZip extracts its files back
// into destDir with their original relative paths and content. This is
// the surface `gormes restore --path` ships to operators who need to
// roll back after a bad update.
func TestRestoreFromZip_ExtractsAllEntries(t *testing.T) {
	srcDir := t.TempDir()
	mustWrite(t, filepath.Join(srcDir, "config.toml"), "[hermes]\nmodel = \"x\"\n")
	mustWrite(t, filepath.Join(srcDir, "skills/SKILL.md"), "skill body\n")
	mustWrite(t, filepath.Join(srcDir, "memory/USER.md"), "user memory\n")

	zipPath := filepath.Join(t.TempDir(), "pre-update-x.zip")
	if _, err := WriteBackupZip(context.Background(), srcDir, zipPath); err != nil {
		t.Fatalf("WriteBackupZip: %v", err)
	}

	dst := t.TempDir()
	if err := RestoreFromZip(context.Background(), zipPath, dst); err != nil {
		t.Fatalf("RestoreFromZip: %v", err)
	}

	for _, want := range []struct {
		rel  string
		body string
	}{
		{"config.toml", "[hermes]\nmodel = \"x\"\n"},
		{"skills/SKILL.md", "skill body\n"},
		{"memory/USER.md", "user memory\n"},
	} {
		full := filepath.Join(dst, want.rel)
		got, err := os.ReadFile(full)
		if err != nil {
			t.Fatalf("read restored %s: %v", want.rel, err)
		}
		if string(got) != want.body {
			t.Fatalf("restored %s body = %q, want %q", want.rel, got, want.body)
		}
	}
}

// TestRestoreFromZip_OverwritesExistingFiles proves the rollback intent:
// a restore intentionally clobbers files in destDir that match zip
// entries. Without this, partial-update damage would survive a
// `gormes restore`.
func TestRestoreFromZip_OverwritesExistingFiles(t *testing.T) {
	srcDir := t.TempDir()
	mustWrite(t, filepath.Join(srcDir, "config.toml"), "[hermes]\nmodel = \"good\"\n")

	zipPath := filepath.Join(t.TempDir(), "pre-update-x.zip")
	if _, err := WriteBackupZip(context.Background(), srcDir, zipPath); err != nil {
		t.Fatalf("WriteBackupZip: %v", err)
	}

	dst := t.TempDir()
	mustWrite(t, filepath.Join(dst, "config.toml"), "[hermes]\nmodel = \"corrupted\"\n")

	if err := RestoreFromZip(context.Background(), zipPath, dst); err != nil {
		t.Fatalf("RestoreFromZip: %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(dst, "config.toml"))
	if !strings.Contains(string(got), "good") {
		t.Fatalf("restore must overwrite existing files; got %q", got)
	}
}

// TestRestoreFromZip_RejectsPathTraversal proves the helper rejects zip
// entries whose names try to escape destDir via `..` or absolute paths.
// A malicious or corrupted zip must not be able to write outside the
// operator-chosen restore root.
func TestRestoreFromZip_RejectsPathTraversal(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "bad.zip")
	for _, evilName := range []string{
		"../escape.txt",
		"a/../../escape.txt",
		"/abs/escape.txt",
	} {
		t.Run(evilName, func(t *testing.T) {
			f, err := os.Create(zipPath)
			if err != nil {
				t.Fatal(err)
			}
			zw := zip.NewWriter(f)
			w, err := zw.Create(evilName)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write([]byte("pwned")); err != nil {
				t.Fatal(err)
			}
			if err := zw.Close(); err != nil {
				t.Fatal(err)
			}
			f.Close()

			err = RestoreFromZip(context.Background(), zipPath, t.TempDir())
			if err == nil {
				t.Fatalf("path-traversal entry %q must be rejected", evilName)
			}
			if !strings.Contains(err.Error(), "path") && !strings.Contains(err.Error(), "traversal") && !strings.Contains(err.Error(), "escape") {
				t.Fatalf("error %q must name the rejection reason", err)
			}
		})
	}
}

// TestRestoreFromZip_PathTraversalRejectsBeforeAnyWrites proves the
// extract is atomic-ish against malicious zips: when a path-traversal
// entry appears AFTER several safe entries, RestoreFromZip must reject
// the whole archive without writing the safe ones first. Otherwise an
// operator who tries to roll back from a half-corrupted zip would end
// up with a partially-restored ~/.gormes — worse than no rollback at
// all.
func TestRestoreFromZip_PathTraversalRejectsBeforeAnyWrites(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "evil.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for _, name := range []string{"safe-1.txt", "skills/safe-2.txt", "memory/safe-3.txt"} {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte("safe-body")); err != nil {
			t.Fatal(err)
		}
	}
	w, err := zw.Create("../escape.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("pwned")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	dst := t.TempDir()
	if err := RestoreFromZip(context.Background(), zipPath, dst); err == nil {
		t.Fatal("malicious zip must be rejected")
	}
	// Walk dst and assert no files were created. The temp dir itself
	// exists (t.TempDir made it) but must be empty.
	entries, err := os.ReadDir(dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if len(entries) != 0 {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Fatalf("dest dir must be empty after rejected restore (atomic-ish); got entries: %v", names)
	}
}

// TestValidateRestoreZip_AcceptsRealBackup proves the validator
// accepts a normal pre-update zip produced by WriteBackupZip — the
// happy path the dry-run preview will exercise before printing "would
// extract".
func TestValidateRestoreZip_AcceptsRealBackup(t *testing.T) {
	srcDir := t.TempDir()
	mustWrite(t, filepath.Join(srcDir, "config.toml"), "x")
	zipPath := filepath.Join(t.TempDir(), "pre-update-x.zip")
	if _, err := WriteBackupZip(context.Background(), srcDir, zipPath); err != nil {
		t.Fatalf("WriteBackupZip: %v", err)
	}
	if err := ValidateRestoreZip(zipPath); err != nil {
		t.Fatalf("valid backup zip must pass validation; got %v", err)
	}
}

// TestValidateRestoreZip_RejectsCorruptArchive proves the validator
// surfaces zip-corruption errors instead of letting the operator
// proceed past the dry-run preview into a half-failed --yes extract.
func TestValidateRestoreZip_RejectsCorruptArchive(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "corrupt.zip")
	if err := os.WriteFile(zipPath, []byte("not a zip file at all"), 0o644); err != nil {
		t.Fatalf("write corrupt fixture: %v", err)
	}
	err := ValidateRestoreZip(zipPath)
	if err == nil {
		t.Fatal("corrupt zip must be rejected")
	}
}

// TestValidateRestoreZip_RejectsPathTraversalEntry proves the validator
// rejects path-traversal zip entries — the same gate RestoreFromZip
// applies at extract time. Surfacing it during dry-run lets operators
// see the rejection before committing to --yes.
func TestValidateRestoreZip_RejectsPathTraversalEntry(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "evil.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("../escape.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("pwned")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	if err := ValidateRestoreZip(zipPath); err == nil {
		t.Fatal("path-traversal entry must be rejected by validator")
	}
}

// TestRestoreFromZip_RejectsMissingSource proves a missing zip file is
// surfaced as a typed error, not a silent no-op. Operators who pass a
// stale path should see the failure.
func TestRestoreFromZip_RejectsMissingSource(t *testing.T) {
	err := RestoreFromZip(context.Background(), filepath.Join(t.TempDir(), "missing.zip"), t.TempDir())
	if err == nil {
		t.Fatal("missing source must error")
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

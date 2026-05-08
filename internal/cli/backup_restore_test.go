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

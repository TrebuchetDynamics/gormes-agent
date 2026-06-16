package restore

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/backup/archive"
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
	if _, err := archive.WriteBackupZip(context.Background(), srcDir, zipPath); err != nil {
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
	if _, err := archive.WriteBackupZip(context.Background(), srcDir, zipPath); err != nil {
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

func TestRestoreFromZip_ReplacesExistingSymlinkInsteadOfFollowingIt(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "pre-update-x.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("config.toml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("restored")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	dst := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	mustWrite(t, outside, "outside-original")
	if err := os.Symlink(outside, filepath.Join(dst, "config.toml")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	if err := RestoreFromZip(context.Background(), zipPath, dst); err != nil {
		t.Fatalf("RestoreFromZip: %v", err)
	}
	outsideBody, err := os.ReadFile(outside)
	if err != nil {
		t.Fatalf("read outside target: %v", err)
	}
	if string(outsideBody) != "outside-original" {
		t.Fatalf("restore followed an existing symlink and overwrote outside file: %q", outsideBody)
	}
	info, err := os.Lstat(filepath.Join(dst, "config.toml"))
	if err != nil {
		t.Fatalf("lstat restored config: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("restore must replace the symlink with a regular file")
	}
	got, err := os.ReadFile(filepath.Join(dst, "config.toml"))
	if err != nil {
		t.Fatalf("read restored config: %v", err)
	}
	if string(got) != "restored" {
		t.Fatalf("restored config = %q, want restored", got)
	}
}

func TestRestoreFromZip_RejectsSymlinkParentBeforeAnyWrites(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "pre-update-x.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for _, entry := range []struct {
		name string
		body string
	}{
		{name: "config.toml", body: "safe-but-must-not-land"},
		{name: "skills/secret.txt", body: "would-escape"},
	} {
		w, err := zw.Create(entry.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(entry.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	dst := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(dst, "skills")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	err = RestoreFromZip(context.Background(), zipPath, dst)
	if err == nil {
		t.Fatal("restore through a symlinked parent must be rejected")
	}
	if !strings.Contains(err.Error(), "symlink") && !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("error must describe symlink escape; got %q", err)
	}
	if _, statErr := os.Stat(filepath.Join(dst, "config.toml")); !os.IsNotExist(statErr) {
		t.Fatalf("restore must reject symlink-parent archive before writing earlier safe entries; stat err = %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(outside, "secret.txt")); !os.IsNotExist(statErr) {
		t.Fatalf("restore must not write through symlinked parent; stat err = %v", statErr)
	}
}

func TestRestoreFromZip_RejectsArchiveFileDirectoryConflictBeforeAnyWrites(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "pre-update-conflict.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("config.toml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("safe-but-must-not-land")); err != nil {
		t.Fatal(err)
	}
	w, err = zw.Create("config.toml/nested.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("conflict")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	dst := t.TempDir()
	err = RestoreFromZip(context.Background(), zipPath, dst)
	if err == nil {
		t.Fatal("file/directory archive conflict must be rejected")
	}
	if !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("error must describe archive conflict; got %q", err)
	}
	if _, statErr := os.Stat(filepath.Join(dst, "config.toml")); !os.IsNotExist(statErr) {
		t.Fatalf("restore must reject conflicting archive before writing earlier entries; stat err = %v", statErr)
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
	if _, err := archive.WriteBackupZip(context.Background(), srcDir, zipPath); err != nil {
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

// TestValidateRestoreZip_DistinguishesMissingFromCorrupt proves the
// validator surfaces operator-friendly error wording: a missing file
// reads as "zip not found: <path>" instead of the double-"open" chain
// that os.Open + fmt.Errorf("open zip: %w") produces by default. A
// corrupted file reads as "zip unreadable: ...". Operators triaging
// `gormes restore --path X` should immediately see WHICH problem they
// hit without parsing nested error layers.
func TestValidateRestoreZip_DistinguishesMissingFromCorrupt(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		err := ValidateRestoreZip(filepath.Join(t.TempDir(), "absent.zip"))
		if err == nil {
			t.Fatal("missing file must error")
		}
		msg := err.Error()
		if !strings.Contains(msg, "zip not found") {
			t.Fatalf("err must say `zip not found`; got %q", msg)
		}
		if strings.Count(msg, "open") > 0 {
			t.Fatalf("err must NOT chain doubled-open wording; got %q", msg)
		}
	})
	t.Run("corrupt file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "corrupt.zip")
		if err := os.WriteFile(path, []byte("not a zip"), 0o644); err != nil {
			t.Fatal(err)
		}
		err := ValidateRestoreZip(path)
		if err == nil {
			t.Fatal("corrupt file must error")
		}
		msg := err.Error()
		if !strings.Contains(msg, "zip unreadable") && !strings.Contains(msg, "not a valid zip") {
			t.Fatalf("err must explain corruption; got %q", msg)
		}
	})
}

func TestValidateRestoreZip_RejectsArchiveFileDirectoryConflict(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "conflict.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("config.toml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("file")); err != nil {
		t.Fatal(err)
	}
	w, err = zw.Create("config.toml/nested.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("nested")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	err = ValidateRestoreZip(zipPath)
	if err == nil {
		t.Fatal("validator must reject file/directory archive conflicts")
	}
	if !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("error must describe archive conflict; got %q", err)
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
// surfaced as a typed error with the operator-friendly "zip not found"
// wording, not the doubled "open ... open ..." chain that os.Open and
// fmt.Errorf would otherwise produce. Operators who pass a stale path
// should see the failure phrased so they can act on it directly.
func TestRestoreFromZip_RejectsMissingSource(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.zip")
	err := RestoreFromZip(context.Background(), missing, t.TempDir())
	if err == nil {
		t.Fatal("missing source must error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "zip not found") {
		t.Fatalf("err must say `zip not found`; got %q", msg)
	}
	if !strings.Contains(msg, missing) {
		t.Fatalf("err must include the offending path; got %q", msg)
	}
}

// TestSummarizeRestoreZipImpact_ClassifiesEntries proves the dry-run
// helper classifies every zip entry as either an overwrite (target
// path already exists in destDir) or a create (target is net-new).
// The dry-run preview surfaces these counts so operators see the
// blast radius before committing to --yes.
func TestSummarizeRestoreZipImpact_ClassifiesEntries(t *testing.T) {
	srcDir := t.TempDir()
	mustWrite(t, filepath.Join(srcDir, "config.toml"), "[hermes]\nmodel = \"x\"\n")
	mustWrite(t, filepath.Join(srcDir, "skills/agent.md"), "skill body\n")
	mustWrite(t, filepath.Join(srcDir, "memory/USER.md"), "user memory\n")

	zipPath := filepath.Join(t.TempDir(), "pre-update-x.zip")
	if _, err := archive.WriteBackupZip(context.Background(), srcDir, zipPath); err != nil {
		t.Fatalf("WriteBackupZip: %v", err)
	}

	dst := t.TempDir()
	// Dest already has config.toml AND skills/agent.md — those overlap
	// the zip and count as overwrites. memory/USER.md is net-new.
	mustWrite(t, filepath.Join(dst, "config.toml"), "stale")
	mustWrite(t, filepath.Join(dst, "skills/agent.md"), "stale")

	impact, err := SummarizeRestoreZipImpact(zipPath, dst)
	if err != nil {
		t.Fatalf("SummarizeRestoreZipImpact: %v", err)
	}
	if impact.Overwrite != 2 {
		t.Fatalf("Overwrite = %d, want 2", impact.Overwrite)
	}
	if impact.Create != 1 {
		t.Fatalf("Create = %d, want 1", impact.Create)
	}
}

// TestSummarizeRestoreZipImpact_RejectsPathTraversal proves the helper
// rejects malicious zip entries with the same gate RestoreFromZip
// applies at extract time. A dry-run preview must NOT report counts
// for a zip that would fail the destructive extract — operators would
// see encouraging numbers and re-run with --yes only to hit the
// rejection.
func TestSummarizeRestoreZipImpact_RejectsPathTraversal(t *testing.T) {
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

	_, err = SummarizeRestoreZipImpact(zipPath, t.TempDir())
	if err == nil {
		t.Fatal("path-traversal entry must be rejected by impact summarizer")
	}
}

func TestSummarizeRestoreZipImpact_RejectsArchiveFileDirectoryConflict(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "conflict.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("config.toml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("file")); err != nil {
		t.Fatal(err)
	}
	w, err = zw.Create("config.toml/nested.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("nested")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = SummarizeRestoreZipImpact(zipPath, t.TempDir())
	if err == nil {
		t.Fatal("impact summarizer must reject file/directory archive conflicts")
	}
	if !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("error must describe archive conflict; got %q", err)
	}
}

// TestWriteBackupZip_PopulatesFileCount proves the writer reports the
// number of regular files actually archived. Surfaced in update
// evidence so operators spot suspiciously thin backups (e.g., a 4MB
// zip containing 3 files vs 200) before relying on it for rollback.
func TestWriteBackupZip_PopulatesFileCount(t *testing.T) {
	srcDir := t.TempDir()
	for _, name := range []string{
		"config.toml",
		"skills/agent.md",
		"memory/USER.md",
		"sessions/2026/05/notes.md",
	} {
		mustWrite(t, filepath.Join(srcDir, name), "body-"+name)
	}

	zipPath := filepath.Join(t.TempDir(), "pre-update-x.zip")
	res, err := archive.WriteBackupZip(context.Background(), srcDir, zipPath)
	if err != nil {
		t.Fatalf("WriteBackupZip: %v", err)
	}
	if res.FileCount != 4 {
		t.Fatalf("FileCount = %d, want 4 regular files", res.FileCount)
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

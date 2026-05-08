package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
)

// TestRestoreCommand_ListPrintsNewestFirst proves `gormes restore
// --list` enumerates pre-update-*.zip files newest-first with size,
// and filters out non-backup files. This is the rollback-discovery
// surface: operators run `--list` to find the right zip for a manual
// restore (the actual extract is a follow-up slice).
func TestRestoreCommand_ListPrintsNewestFirst(t *testing.T) {
	backupsDir := t.TempDir()
	now := time.Now()
	stamps := []string{
		"pre-update-20260501T000000Z.zip",
		"pre-update-20260502T000000Z.zip",
		"pre-update-20260503T000000Z.zip",
	}
	for i, name := range stamps {
		full := filepath.Join(backupsDir, name)
		if err := os.WriteFile(full, []byte("body"+name), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		mt := now.Add(time.Duration(i) * time.Hour)
		if err := os.Chtimes(full, mt, mt); err != nil {
			t.Fatalf("chtimes %s: %v", name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(backupsDir, "NOTES.md"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write NOTES.md: %v", err)
	}

	cmd := newRestoreCommandWithSeams(restoreCommandSeams{
		BackupsDir: func() string { return backupsDir },
	})
	stdout, stderr, err := executeRootCommandForTest(cmd, "--list")
	if err != nil {
		t.Fatalf("restore --list: %v stderr=%s stdout=%s", err, stderr, stdout)
	}

	// All three zip names appear; newest first means 0503 precedes 0501.
	for _, want := range stamps {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	idx0503 := strings.Index(stdout, "pre-update-20260503T000000Z.zip")
	idx0501 := strings.Index(stdout, "pre-update-20260501T000000Z.zip")
	if idx0503 < 0 || idx0501 < 0 || idx0503 >= idx0501 {
		t.Fatalf("newest must precede oldest in output:\n%s", stdout)
	}
	// Operator-owned files must not leak into the listing.
	if strings.Contains(stdout, "NOTES.md") {
		t.Fatalf("NOTES.md must not appear in --list output:\n%s", stdout)
	}
}

// TestRestoreCommand_PathYesExtractsZipIntoGormesHome proves the
// extract path: `gormes restore --path <zip> --yes` unzips entries
// back into GormesHome. Without --yes, the destructive extract is
// gated and the command exits without writing.
func TestRestoreCommand_PathYesExtractsZipIntoGormesHome(t *testing.T) {
	srcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcDir, "config.toml"), []byte("[hermes]\nmodel = \"good\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(t.TempDir(), "pre-update-x.zip")
	if _, err := cli.WriteBackupZip(context.Background(), srcDir, zipPath); err != nil {
		t.Fatalf("WriteBackupZip: %v", err)
	}

	gormesHome := t.TempDir()
	cmd := newRestoreCommandWithSeams(restoreCommandSeams{
		BackupsDir: func() string { return filepath.Dir(zipPath) },
		HomeDir:    func() string { return gormesHome },
	})

	stdout, stderr, err := executeRootCommandForTest(cmd, "--path", zipPath, "--yes")
	if err != nil {
		t.Fatalf("restore --path --yes: %v stderr=%s stdout=%s", err, stderr, stdout)
	}
	got, err := os.ReadFile(filepath.Join(gormesHome, "config.toml"))
	if err != nil {
		t.Fatalf("read restored config.toml: %v", err)
	}
	if !strings.Contains(string(got), "good") {
		t.Fatalf("restored config.toml body = %q, want extracted contents", got)
	}
	if !strings.Contains(stdout, "restored") {
		t.Fatalf("stdout missing success line:\n%s", stdout)
	}
}

// TestRestoreCommand_PathWithoutYesIsDryRun proves that without --yes,
// the destructive extract is gated. Operators get a confirmation
// preview, but no files are written. This matches Hermes' "destructive
// ops require explicit consent" pattern.
func TestRestoreCommand_PathWithoutYesIsDryRun(t *testing.T) {
	srcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcDir, "config.toml"), []byte("from-backup\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(t.TempDir(), "pre-update-x.zip")
	if _, err := cli.WriteBackupZip(context.Background(), srcDir, zipPath); err != nil {
		t.Fatalf("WriteBackupZip: %v", err)
	}

	gormesHome := t.TempDir()
	if err := os.WriteFile(filepath.Join(gormesHome, "config.toml"), []byte("untouched\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newRestoreCommandWithSeams(restoreCommandSeams{
		BackupsDir: func() string { return filepath.Dir(zipPath) },
		HomeDir:    func() string { return gormesHome },
	})
	stdout, stderr, err := executeRootCommandForTest(cmd, "--path", zipPath)
	if err != nil {
		t.Fatalf("restore --path (no --yes): %v stderr=%s", err, stderr)
	}
	got, _ := os.ReadFile(filepath.Join(gormesHome, "config.toml"))
	if string(got) != "untouched\n" {
		t.Fatalf("dry-run must not modify files; got %q, want %q", got, "untouched\n")
	}
	if !strings.Contains(stdout, "--yes") {
		t.Fatalf("dry-run stdout must mention --yes to proceed:\n%s", stdout)
	}
}

// TestRestoreCommand_ListJSONEmitsArray proves `--list --json` emits a
// JSON array of `{path, size_bytes, mod_time}` records sorted
// newest-first. This is the machine-readable surface scripts and
// monitoring use to inventory or reason about backups without parsing
// the human-readable column output.
func TestRestoreCommand_ListJSONEmitsArray(t *testing.T) {
	backupsDir := t.TempDir()
	now := time.Now()
	for i, name := range []string{
		"pre-update-20260501T000000Z.zip",
		"pre-update-20260502T000000Z.zip",
	} {
		full := filepath.Join(backupsDir, name)
		if err := os.WriteFile(full, []byte("body"+name), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		mt := now.Add(time.Duration(i) * time.Hour)
		if err := os.Chtimes(full, mt, mt); err != nil {
			t.Fatalf("chtimes %s: %v", name, err)
		}
	}

	cmd := newRestoreCommandWithSeams(restoreCommandSeams{
		BackupsDir: func() string { return backupsDir },
	})
	stdout, _, err := executeRootCommandForTest(cmd, "--list", "--json")
	if err != nil {
		t.Fatalf("restore --list --json: %v", err)
	}

	var got []struct {
		Path      string    `json:"path"`
		SizeBytes int64     `json:"size_bytes"`
		ModTime   time.Time `json:"mod_time"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2", len(got))
	}
	// Newest first: 20260502 before 20260501.
	if filepath.Base(got[0].Path) != "pre-update-20260502T000000Z.zip" {
		t.Fatalf("got[0].Path = %q, want newest first", got[0].Path)
	}
	if got[0].SizeBytes <= 0 {
		t.Fatalf("got[0].SizeBytes = %d, want > 0", got[0].SizeBytes)
	}
}

// TestRestoreCommand_ListJSONEmptyDirEmitsEmptyArray proves the JSON
// surface stays parseable when no backups exist — a scripting consumer
// gets `[]`, not a free-form "no backups found" message.
func TestRestoreCommand_ListJSONEmptyDirEmitsEmptyArray(t *testing.T) {
	cmd := newRestoreCommandWithSeams(restoreCommandSeams{
		BackupsDir: func() string { return filepath.Join(t.TempDir(), "does-not-exist") },
	})
	stdout, _, err := executeRootCommandForTest(cmd, "--list", "--json")
	if err != nil {
		t.Fatalf("restore --list --json on empty dir: %v", err)
	}
	var got []any
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("empty-dir stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if len(got) != 0 {
		t.Fatalf("empty-dir array must have len=0; got %d", len(got))
	}
}

// TestRestoreCommand_DryRunRejectsCorruptZip proves the dry-run
// validates the zip is openable. A corrupt or non-zip file must NOT
// cause the dry-run to print "would extract" — operators should see
// the failure before they commit to --yes.
func TestRestoreCommand_DryRunRejectsCorruptZip(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "corrupt.zip")
	if err := os.WriteFile(zipPath, []byte("not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newRestoreCommandWithSeams(restoreCommandSeams{
		BackupsDir: func() string { return filepath.Dir(zipPath) },
		HomeDir:    func() string { return t.TempDir() },
	})
	stdout, _, err := executeRootCommandForTest(cmd, "--path", zipPath)
	if err == nil {
		t.Fatalf("dry-run on corrupt zip must error; stdout=%s", stdout)
	}
	if !strings.Contains(err.Error(), "zip") && !strings.Contains(err.Error(), "restore") {
		t.Fatalf("err = %q, want it to name the rejection reason", err)
	}
	if strings.Contains(stdout, "DRY RUN — would extract") {
		t.Fatalf("dry-run on corrupt zip must NOT print would-extract; got:\n%s", stdout)
	}
}

// TestRestoreCommand_DryRunShowsSizeAndAge proves the dry-run preview
// surfaces the same size + age columns operators see in `--list`. A
// blind "would extract /path" line is not enough confidence for a
// destructive op; operators want to confirm the zip's vintage before
// re-running with --yes.
func TestRestoreCommand_DryRunShowsSizeAndAge(t *testing.T) {
	srcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcDir, "config.toml"), []byte(strings.Repeat("x", 5000)), 0o644); err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(t.TempDir(), "pre-update-x.zip")
	if _, err := cli.WriteBackupZip(context.Background(), srcDir, zipPath); err != nil {
		t.Fatalf("WriteBackupZip: %v", err)
	}
	// Backdate the zip so the age column is non-trivial (>1m).
	old := time.Now().Add(-3 * time.Hour)
	if err := os.Chtimes(zipPath, old, old); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	cmd := newRestoreCommandWithSeams(restoreCommandSeams{
		BackupsDir: func() string { return filepath.Dir(zipPath) },
		HomeDir:    func() string { return t.TempDir() },
	})
	stdout, _, err := executeRootCommandForTest(cmd, "--path", zipPath)
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	// The dry-run renders `(<size>, <age>)` with both columns. The
	// `h ago` token is unambiguous; for size we look for the closing
	// `B,` (matches `162B,`, `5.0KB,`, `1.4MB,` while not aliasing
	// path letters).
	if !strings.Contains(stdout, "B, ") {
		t.Fatalf("dry-run must show a human-readable size column like `162B,`; got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "h ago") {
		t.Fatalf("dry-run must show age (e.g. `3h ago`); got:\n%s", stdout)
	}
}

// TestRestoreCommand_LatestYesExtractsNewestZip proves the --latest
// shorthand: among multiple pre-update zips, the one with the newest
// mtime is extracted. Operators in a hurry shouldn't need to look up
// the exact path after a bad update.
func TestRestoreCommand_LatestYesExtractsNewestZip(t *testing.T) {
	backupsDir := t.TempDir()
	gormesHome := t.TempDir()

	now := time.Now()
	for i, marker := range []string{"oldest", "middle", "newest"} {
		srcDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(srcDir, "config.toml"), []byte(marker+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		zipPath := filepath.Join(backupsDir, "pre-update-"+marker+".zip")
		if _, err := cli.WriteBackupZip(context.Background(), srcDir, zipPath); err != nil {
			t.Fatalf("WriteBackupZip %s: %v", marker, err)
		}
		mt := now.Add(time.Duration(i) * time.Hour)
		if err := os.Chtimes(zipPath, mt, mt); err != nil {
			t.Fatalf("chtimes %s: %v", marker, err)
		}
	}

	cmd := newRestoreCommandWithSeams(restoreCommandSeams{
		BackupsDir: func() string { return backupsDir },
		HomeDir:    func() string { return gormesHome },
	})
	stdout, stderr, err := executeRootCommandForTest(cmd, "--latest", "--yes")
	if err != nil {
		t.Fatalf("restore --latest --yes: %v stderr=%s stdout=%s", err, stderr, stdout)
	}
	got, err := os.ReadFile(filepath.Join(gormesHome, "config.toml"))
	if err != nil {
		t.Fatalf("read restored config.toml: %v", err)
	}
	if strings.TrimSpace(string(got)) != "newest" {
		t.Fatalf("--latest must restore newest zip; got %q, want %q", got, "newest")
	}
	if !strings.Contains(stdout, "pre-update-newest.zip") {
		t.Fatalf("stdout must name the resolved zip:\n%s", stdout)
	}
}

// TestRestoreCommand_LatestEmptyDirErrors proves --latest emits a clean
// typed error (not a panic) when no backups exist. Operators who mistype
// the directory or try to restore on a fresh install need to see why.
func TestRestoreCommand_LatestEmptyDirErrors(t *testing.T) {
	cmd := newRestoreCommandWithSeams(restoreCommandSeams{
		BackupsDir: func() string { return filepath.Join(t.TempDir(), "does-not-exist") },
		HomeDir:    func() string { return t.TempDir() },
	})
	_, stderr, err := executeRootCommandForTest(cmd, "--latest", "--yes")
	if err == nil {
		t.Fatalf("--latest with no backups must error; stderr=%s", stderr)
	}
	if !strings.Contains(err.Error(), "no backups") {
		t.Fatalf("error %q must explain the empty-list reason", err)
	}
}

// TestRestoreCommand_ListEmptyDirReportsNoBackups proves a missing or
// empty backups directory yields a quiet "no backups found" line and
// exit 0. Fresh installs hitting `restore --list` should not see an
// error.
func TestRestoreCommand_ListEmptyDirReportsNoBackups(t *testing.T) {
	cmd := newRestoreCommandWithSeams(restoreCommandSeams{
		BackupsDir: func() string { return filepath.Join(t.TempDir(), "does-not-exist") },
	})
	stdout, stderr, err := executeRootCommandForTest(cmd, "--list")
	if err != nil {
		t.Fatalf("restore --list on empty dir: %v stderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "no backups found") {
		t.Fatalf("stdout missing empty-listing copy:\n%s", stdout)
	}
}

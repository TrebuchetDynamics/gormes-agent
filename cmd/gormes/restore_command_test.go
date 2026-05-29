package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
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

// TestRestoreCommand_PathYesJSONEmitsStructuredOutcome proves the
// `--yes --json` path emits a parseable outcome document with build
// provenance, the resolved zip path, the dest root, and the count of
// files actually restored. Operator scripts that drive `gormes restore`
// in automation need a structured outcome to verify what landed —
// scraping the human "restored X into Y" line is fragile.
func TestRestoreCommand_PathYesJSONEmitsStructuredOutcome(t *testing.T) {
	srcDir := t.TempDir()
	for _, name := range []string{"config.toml", "secrets.json", "skills/agent.md"} {
		full := filepath.Join(srcDir, name)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("body-"+name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	zipPath := filepath.Join(t.TempDir(), "pre-update-x.zip")
	if _, err := cli.WriteBackupZip(context.Background(), srcDir, zipPath); err != nil {
		t.Fatalf("WriteBackupZip: %v", err)
	}

	// Dest already has config.toml — that one will overwrite.
	gormesHome := t.TempDir()
	if err := os.WriteFile(filepath.Join(gormesHome, "config.toml"), []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newRestoreCommandWithSeams(restoreCommandSeams{
		BackupsDir: func() string { return filepath.Dir(zipPath) },
		HomeDir:    func() string { return gormesHome },
	})
	stdout, stderr, err := executeRootCommandForTest(cmd, "--path", zipPath, "--yes", "--json")
	if err != nil {
		t.Fatalf("restore --path --yes --json: %v stderr=%s stdout=%s", err, stderr, stdout)
	}
	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Action    string `json:"action"`
		Path      string `json:"path"`
		Dest      string `json:"dest"`
		FileCount int    `json:"file_count"`
		Overwrote int    `json:"overwrote"`
		Created   int    `json:"created"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if got.Build.Version != Version || got.Build.GitCommit == "" {
		t.Fatalf("build provenance missing/wrong: %+v", got.Build)
	}
	if got.Action != "restored" {
		t.Fatalf("got.Action = %q, want %q", got.Action, "restored")
	}
	if got.Path != zipPath {
		t.Fatalf("got.Path = %q, want %q", got.Path, zipPath)
	}
	if got.Dest != gormesHome {
		t.Fatalf("got.Dest = %q, want %q", got.Dest, gormesHome)
	}
	if got.FileCount != 3 {
		t.Fatalf("got.FileCount = %d, want 3 (config.toml + secrets.json + skills/agent.md)", got.FileCount)
	}
	if got.Overwrote != 1 || got.Created != 2 {
		t.Fatalf("got.Overwrote/Created = %d/%d, want 1/2", got.Overwrote, got.Created)
	}
	// JSON mode must not interleave the human "restored X into Y" row,
	// which would make stdout unparseable.
	if strings.Contains(stdout, "restored "+filepath.Base(zipPath)) {
		t.Fatalf("--json must not emit the human row; got:\n%s", stdout)
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

// TestRestoreCommand_ListJSONIncludesBuildProvenance proves
// `gormes restore --list --json` carries the running binary's build
// version + SHA. Same contract as update/doctor/status — captured
// inventory snapshots stay attributable to a specific binary.
func TestRestoreCommand_ListJSONIncludesBuildProvenance(t *testing.T) {
	cmd := newRestoreCommandWithSeams(restoreCommandSeams{
		BackupsDir: func() string { return filepath.Join(t.TempDir(), "does-not-exist") },
	})
	stdout, _, err := executeRootCommandForTest(cmd, "--list", "--json")
	if err != nil {
		t.Fatalf("restore --list --json: %v", err)
	}
	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if got.Build.Version != Version {
		t.Fatalf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Build.GitCommit == "" {
		t.Fatalf("got.build.git_commit must be non-empty")
	}
}

// TestRestoreCommand_ListJSONEmitsArray proves `--list --json` emits a
// `{build, backups: [...]}` document with `{path, size_bytes, mod_time}`
// records sorted newest-first. This is the machine-readable surface
// scripts and monitoring use to inventory or reason about backups without
// parsing the human-readable column output.
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

	var got struct {
		Backups []struct {
			Path      string    `json:"path"`
			SizeBytes int64     `json:"size_bytes"`
			ModTime   time.Time `json:"mod_time"`
		} `json:"backups"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if len(got.Backups) != 2 {
		t.Fatalf("got %d entries, want 2", len(got.Backups))
	}
	// Newest first: 20260502 before 20260501.
	if filepath.Base(got.Backups[0].Path) != "pre-update-20260502T000000Z.zip" {
		t.Fatalf("got.Backups[0].Path = %q, want newest first", got.Backups[0].Path)
	}
	if got.Backups[0].SizeBytes <= 0 {
		t.Fatalf("got.Backups[0].SizeBytes = %d, want > 0", got.Backups[0].SizeBytes)
	}
}

// TestRestoreCommand_ListJSONEmptyDirEmitsEmptyArray proves the JSON
// surface stays parseable when no backups exist — a scripting consumer
// gets `{"build": {...}, "backups": []}`, not a free-form
// "no backups found" message.
func TestRestoreCommand_ListJSONEmptyDirEmitsEmptyArray(t *testing.T) {
	cmd := newRestoreCommandWithSeams(restoreCommandSeams{
		BackupsDir: func() string { return filepath.Join(t.TempDir(), "does-not-exist") },
	})
	stdout, _, err := executeRootCommandForTest(cmd, "--list", "--json")
	if err != nil {
		t.Fatalf("restore --list --json on empty dir: %v", err)
	}
	var got struct {
		Backups []any `json:"backups"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("empty-dir stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if got.Backups == nil {
		t.Fatalf("backups must be `[]`, not omitted/null; got %q", stdout)
	}
	if len(got.Backups) != 0 {
		t.Fatalf("empty-dir backups must have len=0; got %d", len(got.Backups))
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

// TestRestoreCommand_DryRunShowsBlastRadius proves the dry-run preview
// classifies the zip's entries against the destination root: how many
// existing files will be overwritten, how many new files will be
// created. A "would extract X into Y" line is the WHAT and WHERE; the
// blast radius is the HOW MUCH — the missing operator-confidence
// signal before committing to --yes. Without it, operators only learn
// the impact AFTER running the destructive op.
func TestRestoreCommand_DryRunShowsBlastRadius(t *testing.T) {
	// Source directory with three files; one will overlap the dest root.
	srcDir := t.TempDir()
	for _, name := range []string{"config.toml", "secrets.json", "skills/agent.md"} {
		full := filepath.Join(srcDir, name)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("body-"+name), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	zipPath := filepath.Join(t.TempDir(), "pre-update-x.zip")
	if _, err := cli.WriteBackupZip(context.Background(), srcDir, zipPath); err != nil {
		t.Fatalf("WriteBackupZip: %v", err)
	}

	// Dest already has config.toml — that counts as overwrite. The other
	// two zip entries are net-new on the dest tree.
	destHome := t.TempDir()
	if err := os.WriteFile(filepath.Join(destHome, "config.toml"), []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newRestoreCommandWithSeams(restoreCommandSeams{
		BackupsDir: func() string { return filepath.Dir(zipPath) },
		HomeDir:    func() string { return destHome },
	})
	stdout, _, err := executeRootCommandForTest(cmd, "--path", zipPath)
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	for _, want := range []string{
		"would overwrite 1",
		"create 2",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("dry-run must show blast-radius %q; got:\n%s", want, stdout)
		}
	}
}

// TestRestoreCommand_LatestNoBackupsIncludesGuidance proves that when
// `restore --latest` runs against an empty backups directory, the
// error message also tells the operator HOW to create backups (the
// `gormes update --backup` flag or the `[updates] pre_update_backup`
// config key). Without this, operators stuck on a fresh install
// without any backups read "no backups found" and are left to grep
// through docs to figure out which flag enables them.
func TestRestoreCommand_LatestNoBackupsIncludesGuidance(t *testing.T) {
	cmd := newRestoreCommandWithSeams(restoreCommandSeams{
		BackupsDir: func() string { return filepath.Join(t.TempDir(), "no-such-backups") },
		HomeDir:    func() string { return t.TempDir() },
	})
	stdout, _, err := executeRootCommandForTest(cmd, "--latest")
	if err == nil {
		t.Fatalf("--latest with no backups must error; stdout=%s", stdout)
	}
	msg := err.Error()
	if !strings.Contains(msg, "no backups found") {
		t.Fatalf("err must say `no backups found`; got %q", msg)
	}
	if !strings.Contains(msg, "--backup") {
		t.Fatalf("err must hint at `--backup` flag; got %q", msg)
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

// TestRestoreCommand_PathDryRunJSONEmitsStructuredPreview proves
// `restore --path X --json` (without --yes) emits a JSON dry-run
// preview rather than human prose. Operator scripts driving restore
// previews — e.g., to surface "this would clobber 12 files" in CI/CD
// approval gates — need a parseable shape, not log lines. The
// `dry_run: true` flag plus `would_overwrite` / `would_create` lets
// callers distinguish the preview from the executed `restored` shape.
func TestRestoreCommand_PathDryRunJSONEmitsStructuredPreview(t *testing.T) {
	srcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcDir, "config.toml"), []byte("from-backup\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "new-file.toml"), []byte("net-new\n"), 0o644); err != nil {
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
	stdout, stderr, err := executeRootCommandForTest(cmd, "--path", zipPath, "--json")
	if err != nil {
		t.Fatalf("restore --path --json (dry-run): %v stderr=%s", err, stderr)
	}

	// Files MUST stay untouched — JSON dry-run is still a dry-run.
	got, _ := os.ReadFile(filepath.Join(gormesHome, "config.toml"))
	if string(got) != "untouched\n" {
		t.Fatalf("dry-run JSON must not modify files; got %q, want %q", got, "untouched\n")
	}

	var preview struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Action         string `json:"action"`
		Path           string `json:"path"`
		Dest           string `json:"dest"`
		DryRun         bool   `json:"dry_run"`
		WouldOverwrite int    `json:"would_overwrite"`
		WouldCreate    int    `json:"would_create"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &preview); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if preview.Build.Version != Version {
		t.Fatalf("preview.build.version = %q, want %q", preview.Build.Version, Version)
	}
	if preview.Action != "preview" {
		t.Fatalf("preview.action = %q, want %q", preview.Action, "preview")
	}
	if !preview.DryRun {
		t.Fatalf("preview.dry_run must be true; got false")
	}
	if preview.Path != zipPath {
		t.Fatalf("preview.path = %q, want %q", preview.Path, zipPath)
	}
	if preview.Dest != gormesHome {
		t.Fatalf("preview.dest = %q, want %q", preview.Dest, gormesHome)
	}
	if preview.WouldOverwrite != 1 {
		t.Fatalf("preview.would_overwrite = %d, want 1 (config.toml)", preview.WouldOverwrite)
	}
	if preview.WouldCreate != 1 {
		t.Fatalf("preview.would_create = %d, want 1 (new-file.toml)", preview.WouldCreate)
	}
}

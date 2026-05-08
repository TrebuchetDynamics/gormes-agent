package main

import (
	"bytes"
	"encoding/json"
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

// TestCheckpointsCLI_Status_JSONEmitsStructured proves
// `gormes checkpoints status --json` returns a `{build, root, …}`
// document operator scripts can parse to monitor /rollback storage
// growth, identify orphans before pruning, and feed dashboards
// without scraping column-formatted text. Build provenance leads —
// same convention as update/doctor/restore/status `--json`.
func TestCheckpointsCLI_Status_JSONEmitsStructured(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	root := checkpointTestRoot(tmp)
	os.MkdirAll(root, 0o755)

	now := time.Date(2026, 5, 8, 10, 0, 0, 0, time.UTC)
	live := filepath.Join(tmp, "live-project")
	os.MkdirAll(live, 0o755)
	seedCheckpointShadow(t, root, "live-shadow", live, now.Add(-30*time.Minute))
	orphan := filepath.Join(tmp, "deleted-project")
	seedCheckpointShadow(t, root, "orphan-shadow", orphan, now.Add(-2*time.Hour))

	cmd := newCheckpointsCommand()
	cmd.SetArgs([]string{"status", "--json"})
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("checkpoints status --json: %v", err)
	}

	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Root             string `json:"root"`
		TotalSizeBytes   int64  `json:"total_size_bytes"`
		StoreSizeBytes   int64  `json:"store_size_bytes"`
		LegacySizeBytes  int64  `json:"legacy_size_bytes"`
		ProjectCount     int    `json:"project_count"`
		Projects         []struct {
			Name      string    `json:"name"`
			Workdir   string    `json:"workdir"`
			Commits   int       `json:"commits"`
			LastTouch time.Time `json:"last_touch"`
			Exists    bool      `json:"exists"`
		} `json:"projects"`
		LegacyArchives []struct {
			Name      string `json:"name"`
			SizeBytes int64  `json:"size_bytes"`
		} `json:"legacy_archives"`
	}
	if jsonErr := jsonUnmarshalCheckpoints(out.Bytes(), &got); jsonErr != nil {
		t.Fatalf("status --json must be valid JSON: %v\nstdout=%s", jsonErr, out.String())
	}
	if got.Build.Version != Version {
		t.Errorf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Build.GitCommit == "" {
		t.Errorf("got.build.git_commit must be non-empty")
	}
	if got.ProjectCount != 2 {
		t.Errorf("project_count = %d, want 2", got.ProjectCount)
	}
	var sawLive, sawOrphan bool
	for _, p := range got.Projects {
		if p.Workdir == live {
			sawLive = true
			if !p.Exists {
				t.Errorf("live project must report exists=true")
			}
		}
		if p.Workdir == orphan {
			sawOrphan = true
			if p.Exists {
				t.Errorf("orphan project must report exists=false")
			}
		}
	}
	if !sawLive || !sawOrphan {
		t.Errorf("status JSON must include both live and orphan; got %+v", got.Projects)
	}
}

func jsonUnmarshalCheckpoints(b []byte, v any) error {
	return json.Unmarshal(b, v)
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

// TestCheckpointsCLI_Prune_DryRunPreviewsWithoutDeleting proves that
// `gormes checkpoints prune --dry-run` reports the orphan/stale counts
// the operator WOULD lose, but leaves every shadow directory intact on
// disk. Without this, operators can only learn the blast radius by
// actually deleting — which is the very thing they were trying to
// preview. This is symmetric with `update --backup` dry-run and
// `restore --json` preview: destructive ops must offer a non-destructive
// rehearsal.
func TestCheckpointsCLI_Prune_DryRunPreviewsWithoutDeleting(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	root := checkpointTestRoot(tmp)
	os.MkdirAll(root, 0o755)

	now := time.Date(2026, 5, 8, 10, 0, 0, 0, time.UTC)
	orphanWorkdir := filepath.Join(tmp, "deleted-project")
	seedCheckpointShadow(t, root, "orphan-shadow", orphanWorkdir, now.Add(-2*time.Hour))

	staleWorkdir := filepath.Join(tmp, "stale")
	os.MkdirAll(staleWorkdir, 0o755)
	seedCheckpointShadow(t, root, "stale-shadow", staleWorkdir, now.Add(-10*24*time.Hour))

	cmd := newCheckpointsCommand()
	cmd.SetArgs([]string{"prune", "--dry-run", "--retention-days", "5", "--max-size-mb", "500"})
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("checkpoints prune --dry-run: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "DRY RUN") {
		t.Errorf("dry-run output must announce DRY RUN so operators don't mistake the preview for an applied prune\n%s", output)
	}
	// Both shadow dirs MUST still be on disk — that's what makes this a dry run.
	if _, err := os.Stat(filepath.Join(root, "orphan-shadow")); err != nil {
		t.Errorf("orphan-shadow was deleted by --dry-run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "stale-shadow")); err != nil {
		t.Errorf("stale-shadow was deleted by --dry-run: %v", err)
	}
	// Counts still show what WOULD have been pruned.
	if !strings.Contains(output, "Deleted orphan:  1") {
		t.Errorf("dry-run must still tally orphan candidates\n%s", output)
	}
	if !strings.Contains(output, "Deleted stale:   1") {
		t.Errorf("dry-run must still tally stale candidates\n%s", output)
	}
}

// TestCheckpointsCLI_Prune_JSONEmitsStructuredOutcome proves
// `gormes checkpoints prune --json` returns a parseable
// `{build, dry_run, retention_days, max_total_size_mb, keep_orphans,
// scanned, deleted_orphan, deleted_stale, errors, bytes_freed}` shape
// for operator scripts and dashboards. Without machine-readable output,
// scripts have to scrape "Deleted orphan:  N" column text. Dry-run
// applies via `--dry-run` and is reflected in `dry_run: true`.
func TestCheckpointsCLI_Prune_JSONEmitsStructuredOutcome(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	root := checkpointTestRoot(tmp)
	os.MkdirAll(root, 0o755)

	now := time.Date(2026, 5, 8, 10, 0, 0, 0, time.UTC)
	orphanWorkdir := filepath.Join(tmp, "deleted-project")
	seedCheckpointShadow(t, root, "orphan-shadow", orphanWorkdir, now.Add(-2*time.Hour))

	cmd := newCheckpointsCommand()
	cmd.SetArgs([]string{"prune", "--json", "--retention-days", "5", "--max-size-mb", "500"})
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("checkpoints prune --json: %v", err)
	}

	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		DryRun         bool  `json:"dry_run"`
		RetentionDays  int   `json:"retention_days"`
		MaxTotalSizeMB int   `json:"max_total_size_mb"`
		KeepOrphans    bool  `json:"keep_orphans"`
		Scanned        int   `json:"scanned"`
		DeletedOrphan  int   `json:"deleted_orphan"`
		DeletedStale   int   `json:"deleted_stale"`
		Errors         int   `json:"errors"`
		BytesFreed     int64 `json:"bytes_freed"`
	}
	if jsonErr := json.Unmarshal(out.Bytes(), &got); jsonErr != nil {
		t.Fatalf("prune --json must be valid JSON: %v\nstdout=%s", jsonErr, out.String())
	}
	if got.Build.Version != Version {
		t.Errorf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Build.GitCommit == "" {
		t.Errorf("got.build.git_commit must be non-empty")
	}
	if got.DryRun {
		t.Errorf("apply mode must report dry_run=false")
	}
	if got.RetentionDays != 5 {
		t.Errorf("retention_days = %d, want 5", got.RetentionDays)
	}
	if got.MaxTotalSizeMB != 500 {
		t.Errorf("max_total_size_mb = %d, want 500", got.MaxTotalSizeMB)
	}
	if got.DeletedOrphan != 1 {
		t.Errorf("deleted_orphan = %d, want 1", got.DeletedOrphan)
	}
	if _, err := os.Stat(filepath.Join(root, "orphan-shadow")); !os.IsNotExist(err) {
		t.Errorf("orphan-shadow should be gone after apply mode; got err=%v", err)
	}
}

// TestCheckpointsCLI_Prune_DryRunJSONIsReadableFromScripts proves
// `--dry-run --json` reports `dry_run: true` AND keeps the shadow
// directories on disk. Operator scripts gating on prune size before
// pulling the trigger need both signals.
func TestCheckpointsCLI_Prune_DryRunJSONIsReadableFromScripts(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	root := checkpointTestRoot(tmp)
	os.MkdirAll(root, 0o755)

	now := time.Date(2026, 5, 8, 10, 0, 0, 0, time.UTC)
	staleWorkdir := filepath.Join(tmp, "stale")
	os.MkdirAll(staleWorkdir, 0o755)
	seedCheckpointShadow(t, root, "stale-shadow", staleWorkdir, now.Add(-10*24*time.Hour))

	cmd := newCheckpointsCommand()
	cmd.SetArgs([]string{"prune", "--dry-run", "--json", "--retention-days", "5"})
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("checkpoints prune --dry-run --json: %v", err)
	}

	var got struct {
		DryRun       bool `json:"dry_run"`
		DeletedStale int  `json:"deleted_stale"`
	}
	if jsonErr := json.Unmarshal(out.Bytes(), &got); jsonErr != nil {
		t.Fatalf("prune --dry-run --json must be valid JSON: %v\nstdout=%s", jsonErr, out.String())
	}
	if !got.DryRun {
		t.Errorf("dry-run mode must report dry_run=true; got=%+v", got)
	}
	if got.DeletedStale != 1 {
		t.Errorf("deleted_stale = %d, want 1 (count still reflects what WOULD have been pruned)", got.DeletedStale)
	}
	if _, err := os.Stat(filepath.Join(root, "stale-shadow")); err != nil {
		t.Errorf("stale-shadow must remain on disk in dry-run; got err=%v", err)
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

// TestCheckpointsCLI_Clear_JSONEmitsStructuredOutcome proves
// `gormes checkpoints clear -f --json` returns
// `{build, root, action, deleted, bytes_freed, projects_before,
// legacy_before}` so operator scripts performing scheduled GC can
// audit/log what was destroyed without scraping prose. Because clear
// is total destruction, the pre-state (projects/legacy counts before
// the wipe) is captured BEFORE delete and embedded in the JSON.
func TestCheckpointsCLI_Clear_JSONEmitsStructuredOutcome(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	root := checkpointTestRoot(tmp)
	os.MkdirAll(root, 0o755)

	now := time.Date(2026, 5, 8, 10, 0, 0, 0, time.UTC)
	wd := filepath.Join(tmp, "live")
	os.MkdirAll(wd, 0o755)
	seedCheckpointShadow(t, root, "live", wd, now)
	seedLegacyArchive(t, root, "legacy-1")

	cmd := newCheckpointsCommand()
	cmd.SetArgs([]string{"clear", "-f", "--json"})
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("checkpoints clear -f --json: %v", err)
	}

	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Root           string `json:"root"`
		Action         string `json:"action"`
		Deleted        bool   `json:"deleted"`
		BytesFreed     int64  `json:"bytes_freed"`
		ProjectsBefore int    `json:"projects_before"`
		LegacyBefore   int    `json:"legacy_before"`
	}
	if jsonErr := json.Unmarshal(out.Bytes(), &got); jsonErr != nil {
		t.Fatalf("clear --json must be valid JSON: %v\nstdout=%s", jsonErr, out.String())
	}
	if got.Build.Version != Version {
		t.Errorf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Action != "cleared" {
		t.Errorf("action = %q, want %q", got.Action, "cleared")
	}
	if !got.Deleted {
		t.Errorf("deleted must be true after clear")
	}
	if got.ProjectsBefore != 1 {
		t.Errorf("projects_before = %d, want 1 (the live shadow seeded into root)", got.ProjectsBefore)
	}
	if got.LegacyBefore != 1 {
		t.Errorf("legacy_before = %d, want 1 (the legacy-1 archive)", got.LegacyBefore)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Errorf("clear must remove the root; stat err=%v", err)
	}
}

// TestCheckpointsCLI_ClearLegacy_JSONEmitsStructuredOutcome proves
// `gormes checkpoints clear-legacy -f --json` returns
// `{build, root, action, archives_before: [...], deleted, bytes_freed}`
// for operator scripts cleaning up post-migration v1 shadow repos.
// `archives_before` lists the names+sizes of what was destroyed —
// captured BEFORE delete so the JSON consumer learns exactly which
// archives are gone.
func TestCheckpointsCLI_ClearLegacy_JSONEmitsStructuredOutcome(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	root := checkpointTestRoot(tmp)
	os.MkdirAll(root, 0o755)
	seedLegacyArchive(t, root, "legacy-1")
	seedLegacyArchive(t, root, "legacy-2")

	cmd := newCheckpointsCommand()
	cmd.SetArgs([]string{"clear-legacy", "-f", "--json"})
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("checkpoints clear-legacy -f --json: %v", err)
	}

	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Root            string `json:"root"`
		Action          string `json:"action"`
		ArchivesBefore  []struct {
			Name      string `json:"name"`
			SizeBytes int64  `json:"size_bytes"`
		} `json:"archives_before"`
		Deleted    int   `json:"deleted"`
		BytesFreed int64 `json:"bytes_freed"`
	}
	if jsonErr := json.Unmarshal(out.Bytes(), &got); jsonErr != nil {
		t.Fatalf("clear-legacy --json must be valid JSON: %v\nstdout=%s", jsonErr, out.String())
	}
	if got.Build.Version != Version {
		t.Errorf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Action != "cleared" {
		t.Errorf("action = %q, want %q", got.Action, "cleared")
	}
	if got.Deleted != 2 {
		t.Errorf("deleted = %d, want 2 (the two seeded legacy archives)", got.Deleted)
	}
	if len(got.ArchivesBefore) != 2 {
		t.Errorf("archives_before len = %d, want 2", len(got.ArchivesBefore))
	}
	for _, e := range []string{"legacy-1", "legacy-2"} {
		var saw bool
		for _, a := range got.ArchivesBefore {
			if a.Name == e {
				saw = true
				break
			}
		}
		if !saw {
			t.Errorf("archives_before missing %q; got %+v", e, got.ArchivesBefore)
		}
	}
	for _, name := range []string{"legacy-1", "legacy-2"} {
		if _, err := os.Stat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Errorf("%s should be gone after clear-legacy; stat err=%v", name, err)
		}
	}
}

// TestCheckpointsCLI_ClearLegacy_JSONNoopWhenEmpty proves that when
// no legacy archives exist, `clear-legacy --json` emits a
// `{action: "noop"}` shape rather than scraping "No legacy archives
// to clear." Operators on fresh installs need a stable JSON contract.
func TestCheckpointsCLI_ClearLegacy_JSONNoopWhenEmpty(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	root := checkpointTestRoot(tmp)
	os.MkdirAll(root, 0o755)

	cmd := newCheckpointsCommand()
	cmd.SetArgs([]string{"clear-legacy", "-f", "--json"})
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("checkpoints clear-legacy -f --json (empty): %v", err)
	}

	var got struct {
		Action  string `json:"action"`
		Deleted int    `json:"deleted"`
	}
	if jsonErr := json.Unmarshal(out.Bytes(), &got); jsonErr != nil {
		t.Fatalf("clear-legacy --json (empty) must be valid JSON: %v\nstdout=%s", jsonErr, out.String())
	}
	if got.Action != "noop" {
		t.Errorf("action = %q, want %q", got.Action, "noop")
	}
	if got.Deleted != 0 {
		t.Errorf("deleted = %d, want 0", got.Deleted)
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

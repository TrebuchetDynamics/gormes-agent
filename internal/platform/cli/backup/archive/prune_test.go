package archive

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

// TestPruneBackups_KeepsNewestN proves pruning keeps the newest `keep`
// files by mtime and removes the rest. Newest is preserved no matter
// how many older files exist.
func TestPruneBackups_KeepsNewestN(t *testing.T) {
	dir := t.TempDir()
	// Create 5 fixtures with monotonically older mtimes.
	now := time.Now()
	stamps := []string{
		"pre-update-20260501T000000Z.zip",
		"pre-update-20260502T000000Z.zip",
		"pre-update-20260503T000000Z.zip",
		"pre-update-20260504T000000Z.zip",
		"pre-update-20260505T000000Z.zip",
	}
	for i, name := range stamps {
		full := filepath.Join(dir, name)
		if err := os.WriteFile(full, []byte("body"+name), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		mt := now.Add(time.Duration(i) * time.Hour)
		if err := os.Chtimes(full, mt, mt); err != nil {
			t.Fatalf("chtimes %s: %v", name, err)
		}
	}

	count, freed, err := PruneBackups(dir, 2)
	if err != nil {
		t.Fatalf("PruneBackups: %v", err)
	}
	if count != 3 {
		t.Fatalf("removed count = %d, want 3", count)
	}
	if freed <= 0 {
		t.Fatalf("freed bytes must be > 0; got %d", freed)
	}

	remaining := listZipNames(t, dir)
	sort.Strings(remaining)
	want := []string{
		"pre-update-20260504T000000Z.zip",
		"pre-update-20260505T000000Z.zip",
	}
	if len(remaining) != len(want) {
		t.Fatalf("remaining = %v, want %v", remaining, want)
	}
	for i, w := range want {
		if remaining[i] != w {
			t.Fatalf("remaining[%d] = %q, want %q", i, remaining[i], w)
		}
	}
}

// TestPruneBackups_NoOpWhenAtOrUnderKeep proves pruning is a no-op when
// the directory has fewer files than the keep target.
func TestPruneBackups_NoOpWhenAtOrUnderKeep(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pre-update-x.zip"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	count, freed, err := PruneBackups(dir, 5)
	if err != nil {
		t.Fatalf("PruneBackups: %v", err)
	}
	if count != 0 || freed != 0 {
		t.Fatalf("under-keep prune must be no-op; got count=%d freed=%d", count, freed)
	}
}

// TestPruneBackups_IgnoresNonBackupFiles proves the helper only touches
// files matching the pre-update-*.zip pattern. Stray operator files in
// the same directory must not be deleted.
func TestPruneBackups_IgnoresNonBackupFiles(t *testing.T) {
	dir := t.TempDir()
	keep := []string{"NOTES.md", "manifest.json", "other-archive.tar.gz"}
	for _, n := range keep {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644); err != nil {
			t.Fatalf("write %s: %v", n, err)
		}
	}
	// Plus 3 backup files that should be pruned to 1.
	now := time.Now()
	for i, n := range []string{"pre-update-a.zip", "pre-update-b.zip", "pre-update-c.zip"} {
		full := filepath.Join(dir, n)
		if err := os.WriteFile(full, []byte("body"), 0o644); err != nil {
			t.Fatalf("write %s: %v", n, err)
		}
		mt := now.Add(time.Duration(i) * time.Hour)
		if err := os.Chtimes(full, mt, mt); err != nil {
			t.Fatalf("chtimes %s: %v", n, err)
		}
	}
	if _, _, err := PruneBackups(dir, 1); err != nil {
		t.Fatalf("PruneBackups: %v", err)
	}
	// Operator files must survive.
	for _, n := range keep {
		if _, err := os.Stat(filepath.Join(dir, n)); err != nil {
			t.Fatalf("operator file %s must not be deleted; stat err = %v", n, err)
		}
	}
}

// TestPruneBackups_KeepZeroIsNoOpSafety proves keep=0 (and keep<0) is
// treated as a safety no-op rather than wiping the entire directory.
// Operators who passed --backup-keep 0 by mistake should not lose data.
func TestPruneBackups_KeepZeroIsNoOpSafety(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pre-update-x.zip"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	for _, k := range []int{0, -1} {
		count, _, err := PruneBackups(dir, k)
		if err != nil {
			t.Fatalf("keep=%d returned err: %v", k, err)
		}
		if count != 0 {
			t.Fatalf("keep=%d must be no-op safety; got count=%d", k, count)
		}
		if _, err := os.Stat(filepath.Join(dir, "pre-update-x.zip")); err != nil {
			t.Fatalf("file must survive keep=%d safety; stat err = %v", k, err)
		}
	}
}

// TestPruneBackups_MissingDirReturnsNoOp proves a non-existent dir is a
// no-op rather than an error (a fresh install with no prior backups
// should not error during the post-write prune).
func TestPruneBackups_MissingDirReturnsNoOp(t *testing.T) {
	count, freed, err := PruneBackups(filepath.Join(t.TempDir(), "does-not-exist"), 5)
	if err != nil {
		t.Fatalf("missing dir must be no-op; got err = %v", err)
	}
	if count != 0 || freed != 0 {
		t.Fatalf("missing dir must yield zero counts; got count=%d freed=%d", count, freed)
	}
}

// TestListBackups_NewestFirst proves the listing helper returns
// pre-update-*.zip files sorted by mtime (newest first), with stat
// metadata populated. Operators see this list through `gormes restore
// --list` and need the newest one at the top so the first row is the
// most likely rollback target.
func TestListBackups_NewestFirst(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	stamps := []string{
		"pre-update-20260501T000000Z.zip",
		"pre-update-20260502T000000Z.zip",
		"pre-update-20260503T000000Z.zip",
	}
	for i, name := range stamps {
		full := filepath.Join(dir, name)
		if err := os.WriteFile(full, []byte("body"+name), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		mt := now.Add(time.Duration(i) * time.Hour)
		if err := os.Chtimes(full, mt, mt); err != nil {
			t.Fatalf("chtimes %s: %v", name, err)
		}
	}
	// Drop a stray operator-owned file that must not appear in the list.
	if err := os.WriteFile(filepath.Join(dir, "NOTES.md"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write NOTES.md: %v", err)
	}

	got, err := ListBackups(dir)
	if err != nil {
		t.Fatalf("ListBackups: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("ListBackups returned %d entries, want 3 (operator file must be filtered)", len(got))
	}
	wantOrder := []string{
		"pre-update-20260503T000000Z.zip",
		"pre-update-20260502T000000Z.zip",
		"pre-update-20260501T000000Z.zip",
	}
	for i, want := range wantOrder {
		if filepath.Base(got[i].Path) != want {
			t.Fatalf("got[%d].Path = %q, want %q (newest first)", i, filepath.Base(got[i].Path), want)
		}
		if got[i].SizeBytes <= 0 {
			t.Fatalf("got[%d].SizeBytes = %d, want > 0", i, got[i].SizeBytes)
		}
		if got[i].ModTime.IsZero() {
			t.Fatalf("got[%d].ModTime is zero, want stat mtime", i)
		}
	}
}

// TestListBackups_MissingDirIsEmpty proves a non-existent backups dir
// is a quiet empty-list (matches PruneBackups' fresh-install no-op).
// Operators on a fresh install hitting `gormes restore --list` should
// see "no backups found", not an error.
func TestListBackups_MissingDirIsEmpty(t *testing.T) {
	got, err := ListBackups(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("missing dir must be no-op; got err = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("missing dir must yield empty list; got %d entries", len(got))
	}
}

func listZipNames(t *testing.T, dir string) []string {
	t.Helper()
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	out := make([]string, 0, len(ents))
	for _, e := range ents {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".zip" {
			out = append(out, e.Name())
		}
	}
	return out
}

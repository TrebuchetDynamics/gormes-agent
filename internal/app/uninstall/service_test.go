package uninstall

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSortedExistingDedupesSortsAndMarksDirectories(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	subdir := filepath.Join(dir, "dir")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(subdir, 0o700); err != nil {
		t.Fatal(err)
	}

	got := SortedExisting("", subdir, file, file, filepath.Join(dir, "missing"))
	want := []string{filepath.Join(dir, "dir") + "/", filepath.Join(dir, "file")}
	if len(got) != len(want) {
		t.Fatalf("SortedExisting length = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SortedExisting[%d] = %q, want %q (all %#v)", i, got[i], want[i], got)
		}
	}
}

func TestRemoveGroup(t *testing.T) {
	groups := []ArtifactGroup{{Name: "config"}, {Name: "credentials"}, {Name: "sessions"}}
	got := RemoveGroup(groups, "credentials")
	if len(got) != 2 || got[0].Name != "config" || got[1].Name != "sessions" {
		t.Fatalf("RemoveGroup returned %#v", got)
	}
}

func TestCollectPublishedBinaryPathsOnlySymlinksIntoHome(t *testing.T) {
	t.Setenv("GORMES_PREFIX", "")
	t.Setenv("HOME", t.TempDir())
	home := t.TempDir()
	bin := t.TempDir()
	t.Setenv("GORMES_BIN_DIR", bin)

	managed := filepath.Join(home, "bin", "gormes")
	if err := os.MkdirAll(filepath.Dir(managed), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(managed, []byte("bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(bin, "gormes")
	if err := os.Symlink(managed, link); err != nil {
		t.Fatal(err)
	}

	got := CollectPublishedBinaryPaths(home)
	if len(got) != 1 || got[0] != link {
		t.Fatalf("CollectPublishedBinaryPaths = %#v, want [%q]", got, link)
	}
}

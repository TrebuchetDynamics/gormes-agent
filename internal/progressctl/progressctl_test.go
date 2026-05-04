package progressctl

import (
	"path/filepath"
	"testing"
)

func TestProgressPathsMirrorSiteProgressToTrackedDataFiles(t *testing.T) {
	root := t.TempDir()
	paths := progressPaths(root)

	want := []string{
		filepath.Join(root, "webpages", "landing", "src", "data", "progress.json"),
		filepath.Join(root, "webpages", "landing", "legacy", "go-renderer", "internal", "site", "data", "progress.json"),
	}
	if len(paths.siteProgress) != len(want) {
		t.Fatalf("site progress paths = %v, want %v", paths.siteProgress, want)
	}
	for i, path := range want {
		if paths.siteProgress[i] != path {
			t.Fatalf("site progress path[%d] = %q, want %q", i, paths.siteProgress[i], path)
		}
	}

	obsolete := filepath.Join(root, "webpages", "landing", "internal", "site", "data", "progress.json")
	for _, path := range paths.siteProgress {
		if path == obsolete {
			t.Fatalf("site progress paths still include obsolete untracked path %q", obsolete)
		}
	}
}

package progressctl

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
)

// TestValidateJSON_EmitsStatsAndCounts proves that
// `progress validate --format json` emits a parseable
// `{ok, phases, subphases: {total, ...}, items: {total, ...}}`
// document so CI dashboards and fleet monitoring can ingest the
// roadmap status without parsing the human-readable line.
func TestValidateJSON_EmitsStatsAndCounts(t *testing.T) {
	root := repoRootForTest(t)
	var buf bytes.Buffer
	if err := Validate(&buf, root, "json"); err != nil {
		t.Fatalf("Validate json: %v", err)
	}
	var got struct {
		OK        bool `json:"ok"`
		Phases    int  `json:"phases"`
		Subphases struct {
			Total      int `json:"total"`
			Complete   int `json:"complete"`
			InProgress int `json:"in_progress"`
			Planned    int `json:"planned"`
		} `json:"subphases"`
		Items struct {
			Total      int `json:"total"`
			Complete   int `json:"complete"`
			InProgress int `json:"in_progress"`
			Planned    int `json:"planned"`
		} `json:"items"`
	}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\nbody=%s", err, buf.String())
	}
	if !got.OK {
		t.Errorf("ok = false, want true")
	}
	if got.Phases < 6 {
		t.Errorf("phases = %d, want >= 6 (still addressable)", got.Phases)
	}
	if got.Subphases.Total < 50 {
		t.Errorf("subphases.total = %d, want >= 50 (floor catch-all)", got.Subphases.Total)
	}
	if got.Items.Total < 100 {
		t.Errorf("items.total = %d, want >= 100", got.Items.Total)
	}
	if got.Subphases.Complete+got.Subphases.InProgress+got.Subphases.Planned > got.Subphases.Total {
		t.Errorf("subphase derived counts (%d+%d+%d) exceed total %d",
			got.Subphases.Complete, got.Subphases.InProgress, got.Subphases.Planned, got.Subphases.Total)
	}
}

func repoRootForTest(t *testing.T) string {
	t.Helper()
	wd := filepath.Join("..", "..")
	abs, err := filepath.Abs(wd)
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	return abs
}

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

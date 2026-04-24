package cli

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// The upstream hermes_cli/ tree (Phase 5.O) has exactly 49 Python files
// (excluding the example YAML). The Go port tracks every one of those
// files via a typed manifest so a single per-file port never silently
// drops an upstream source. See `testdata/hermes_cli_tree.json`.
const expectedHermesCLIFileCount = 49

// hermesCLIManifestPath is resolved from internal/cli/ (test cwd).
const hermesCLIManifestPath = "../../testdata/hermes_cli_tree.json"

func loadManifestOrFatal(t *testing.T) HermesCLITree {
	t.Helper()
	tree, err := LoadHermesCLITree(hermesCLIManifestPath)
	if err != nil {
		t.Fatalf("load hermes cli tree: %v", err)
	}
	return tree
}

func TestLoadHermesCLITree_CountMatchesUpstream(t *testing.T) {
	tree := loadManifestOrFatal(t)
	if got := len(tree.Files); got != expectedHermesCLIFileCount {
		t.Fatalf("hermes cli manifest entry count = %d, want %d", got, expectedHermesCLIFileCount)
	}
}

func TestLoadHermesCLITree_NoDuplicateSources(t *testing.T) {
	tree := loadManifestOrFatal(t)
	seen := make(map[string]struct{}, len(tree.Files))
	for _, entry := range tree.Files {
		if _, dup := seen[entry.Source]; dup {
			t.Fatalf("duplicate hermes_cli source %q in manifest", entry.Source)
		}
		seen[entry.Source] = struct{}{}
	}
}

func TestLoadHermesCLITree_EntriesAreValid(t *testing.T) {
	tree := loadManifestOrFatal(t)
	valid := map[HermesCLIPortStatus]struct{}{
		HermesCLIPortStatusPorted:        {},
		HermesCLIPortStatusInProgress:    {},
		HermesCLIPortStatusPlanned:       {},
		HermesCLIPortStatusNotApplicable: {},
	}
	for _, entry := range tree.Files {
		if entry.Source == "" {
			t.Fatalf("manifest entry with empty Source: %+v", entry)
		}
		if filepath.Ext(entry.Source) != ".py" {
			t.Fatalf("manifest entry Source %q must be a Python file", entry.Source)
		}
		if _, ok := valid[entry.Status]; !ok {
			t.Fatalf("manifest entry %q has unknown status %q", entry.Source, entry.Status)
		}
		// "ported" / "in_progress" entries must name at least one Go destination.
		needsGo := entry.Status == HermesCLIPortStatusPorted || entry.Status == HermesCLIPortStatusInProgress
		if needsGo && len(entry.Go) == 0 {
			t.Fatalf("manifest entry %q has status %q but no Go destination", entry.Source, entry.Status)
		}
		if !needsGo && len(entry.Go) > 0 {
			t.Fatalf("manifest entry %q has status %q but lists Go destinations %v", entry.Source, entry.Status, entry.Go)
		}
	}
}

func TestHermesCLITree_GoDestinationsExist(t *testing.T) {
	tree := loadManifestOrFatal(t)
	// Tests run from internal/cli/, so module root is two levels up.
	moduleRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("resolve module root: %v", err)
	}
	for _, entry := range tree.Files {
		for _, rel := range entry.Go {
			if filepath.IsAbs(rel) {
				t.Fatalf("manifest %q Go destination %q must be module-relative", entry.Source, rel)
			}
			full := filepath.Join(moduleRoot, rel)
			info, err := os.Stat(full)
			if err != nil {
				t.Fatalf("manifest %q references missing Go destination %q: %v", entry.Source, rel, err)
			}
			if info.IsDir() {
				// A directory is acceptable for subsystem-wide ports (e.g. internal/gateway).
				continue
			}
		}
	}
}

func TestHermesCLITree_CoversEveryUpstreamFile(t *testing.T) {
	// Ensure the manifest covers exactly the real on-disk Python files.
	tree := loadManifestOrFatal(t)
	manifest := make(map[string]struct{}, len(tree.Files))
	for _, entry := range tree.Files {
		manifest[entry.Source] = struct{}{}
	}

	// The upstream Python tree sits next to gormes/ at worker2/hermes_cli/.
	// Locate it from this test's cwd (gormes/internal/cli/).
	upstream, err := filepath.Abs("../../../hermes_cli")
	if err != nil {
		t.Fatalf("resolve upstream hermes_cli: %v", err)
	}
	entries, err := os.ReadDir(upstream)
	if err != nil {
		t.Skipf("upstream hermes_cli/ tree not available (%v) — manifest still enforces count invariant", err)
	}
	var found []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if filepath.Ext(name) != ".py" {
			continue
		}
		found = append(found, name)
	}
	sort.Strings(found)
	if len(found) != expectedHermesCLIFileCount {
		t.Fatalf("upstream hermes_cli/ has %d .py files, expected %d; adjust the manifest", len(found), expectedHermesCLIFileCount)
	}
	for _, name := range found {
		if _, ok := manifest[name]; !ok {
			t.Fatalf("upstream hermes_cli/%s missing from manifest", name)
		}
	}
}

func TestHermesCLITree_SummaryCounts(t *testing.T) {
	tree := loadManifestOrFatal(t)
	summary := tree.Summary()
	if summary.Total != expectedHermesCLIFileCount {
		t.Fatalf("summary total = %d, want %d", summary.Total, expectedHermesCLIFileCount)
	}
	sum := summary.Ported + summary.InProgress + summary.Planned + summary.NotApplicable
	if sum != summary.Total {
		t.Fatalf("summary bucket sum = %d, want %d", sum, summary.Total)
	}
	// At least one entry must be ported — otherwise the manifest is stale,
	// because subsystems like gateway, cron, auth, doctor, memory, skills,
	// plugins, mcp, and acp already have Go ports in this repo.
	if summary.Ported == 0 {
		t.Fatal("summary reports zero ported entries; manifest is stale")
	}
}

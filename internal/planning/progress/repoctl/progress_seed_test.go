package repoctl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress"
)

func TestSeedProgressRowsAddsFleetRowsIdempotently(t *testing.T) {
	root := t.TempDir()
	progressPath := filepath.Join(root, "docs", "content", "building-gormes", "architecture_plan", "progress.json")
	writeSeedProgressFixture(t, progressPath)

	first, err := SeedProgressRows(ProgressSeedOptions{Root: root, Set: "fleet"})
	if err != nil {
		t.Fatalf("SeedProgressRows first: %v", err)
	}
	if first.Added == 0 {
		t.Fatalf("first Added = 0, want fleet rows added: %+v", first)
	}
	if !containsString(first.AddedNames, "Sandbox Policy Explain") {
		t.Fatalf("fleet seed did not add Sandbox Policy Explain: %+v", first.AddedNames)
	}

	second, err := SeedProgressRows(ProgressSeedOptions{Root: root, Set: "fleet"})
	if err != nil {
		t.Fatalf("SeedProgressRows second: %v", err)
	}
	if second.Added != 0 || second.Skipped == 0 {
		t.Fatalf("second result = %+v, want idempotent skip-only run", second)
	}

	p, err := progress.Load(progressPath)
	if err != nil {
		t.Fatalf("load seeded progress: %v", err)
	}
	if err := progress.Validate(p); err != nil {
		t.Fatalf("seeded progress should validate: %v", err)
	}
}

func TestSeedProgressRowsRejectsUnknownSet(t *testing.T) {
	_, err := SeedProgressRows(ProgressSeedOptions{Root: t.TempDir(), Set: "missing"})
	if err == nil || !strings.Contains(err.Error(), "unknown progress row seed set") {
		t.Fatalf("SeedProgressRows unknown set error = %v", err)
	}
}

func writeSeedProgressFixture(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	p := &progress.Progress{
		Meta: progress.Meta{Version: "2.0"},
		Phases: map[string]progress.Phase{
			"5": {
				Name:        "Phase 5",
				Deliverable: "fixture",
				Subphases: map[string]progress.Subphase{
					"5.B": {Name: "Sandbox", Status: progress.StatusPlanned},
					"5.I": {Name: "Plugins", Status: progress.StatusPlanned},
					"5.N": {Name: "Tools", Status: progress.StatusPlanned},
					"5.O": {Name: "Onboarding", Status: progress.StatusPlanned},
				},
			},
		},
	}
	if err := progress.SaveProgress(path, p); err != nil {
		t.Fatalf("SaveProgress fixture: %v", err)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

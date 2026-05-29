package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress"
)

func TestRunProgressSeedRoutesToSeedCatalog(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run(&stdout, &stderr, []string{"--repo-root", t.TempDir(), "progress", "seed", "fleet"})
	if err == nil || !strings.Contains(err.Error(), "progress: read") {
		t.Fatalf("run progress seed error = %v, want progress load error proving route exists", err)
	}
}

func TestRunHermesContractInventoryWritesReports(t *testing.T) {
	root := t.TempDir()
	progressPath := filepath.Join(root, "webpages/docs/content/building-gormes/architecture_plan/progress.json")
	if err := os.MkdirAll(filepath.Dir(progressPath), 0o755); err != nil {
		t.Fatalf("mkdir progress: %v", err)
	}
	if err := progress.SaveProgress(progressPath, &progress.Progress{
		Meta:   progress.Meta{Version: "test"},
		Phases: map[string]progress.Phase{},
	}); err != nil {
		t.Fatalf("write progress: %v", err)
	}

	var stdout, stderr bytes.Buffer
	err := run(&stdout, &stderr, []string{"--repo-root", root, "hermes-contract-inventory", "--hermes-sha", "abc123"})
	if err != nil {
		t.Fatalf("run hermes-contract-inventory error = %v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "repoctl: hermes-contract-inventory report updated") {
		t.Fatalf("stdout = %q, want inventory summary", stdout.String())
	}
	for _, rel := range []string{
		"webpages/docs/content/building-gormes/architecture_plan/hermes-contract-inventory.json",
		"webpages/docs/content/building-gormes/architecture_plan/hermes-contract-inventory.md",
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected report %s: %v", rel, err)
		}
	}
}

func TestRunHermesContractInventoryStrictFailsAfterWritingReports(t *testing.T) {
	root := t.TempDir()
	progressPath := filepath.Join(root, "webpages/docs/content/building-gormes/architecture_plan/progress.json")
	if err := os.MkdirAll(filepath.Dir(progressPath), 0o755); err != nil {
		t.Fatalf("mkdir progress: %v", err)
	}
	if err := progress.SaveProgress(progressPath, &progress.Progress{
		Meta:   progress.Meta{Version: "test"},
		Phases: map[string]progress.Phase{},
	}); err != nil {
		t.Fatalf("write progress: %v", err)
	}

	var stdout, stderr bytes.Buffer
	err := run(&stdout, &stderr, []string{"--repo-root", root, "hermes-contract-inventory", "--hermes-sha", "abc123", "--strict"})
	if err == nil || !strings.Contains(err.Error(), "strict failed") {
		t.Fatalf("strict error = %v, want strict failure\nstdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "repoctl: hermes-contract-inventory report updated") {
		t.Fatalf("strict stdout = %q, want report written before failure", stdout.String())
	}
}

func TestRunHermesSourcePairsValidateRoutesToManifest(t *testing.T) {
	root := t.TempDir()
	for _, path := range []string{
		"hermes-agent/hermes_cli/default_soul.py",
		"internal/hermes/default_soul.go",
		"docs/content/building-gormes/architecture_plan/hermes-source-pairs.json",
	} {
		abs := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
	}
	manifest := `{
  "schema_version": "1.0",
  "hermes_sha": "abc123",
  "pairs": [
    {
      "hermes_file": "hermes_cli/default_soul.py",
      "gormes_targets": ["internal/hermes/default_soul.go"],
      "status": "covered",
      "contract": "Default SOUL.md seed text",
      "tests": ["go test ./internal/hermes -run TestDefaultSoul -count=1"],
      "last_checked_hermes_sha": "abc123"
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(root, "docs/content/building-gormes/architecture_plan/hermes-source-pairs.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "hermes-agent/hermes_cli/default_soul.py"), []byte("fixture\n"), 0o644); err != nil {
		t.Fatalf("write hermes file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal/hermes/default_soul.go"), []byte("fixture\n"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	var stdout, stderr bytes.Buffer
	err := run(&stdout, &stderr, []string{"--repo-root", root, "hermes-source-pairs", "validate", "--allow-unmapped-high-priority", "--hermes-sha", "abc123"})
	if err != nil {
		t.Fatalf("run validate error = %v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "repoctl: hermes-source-pairs validate ok pairs=1") {
		t.Fatalf("stdout = %q, want validate summary", stdout.String())
	}
}

func TestRunHermesSourcePairsSyncSHARoutesToManifest(t *testing.T) {
	root := t.TempDir()
	for _, path := range []string{
		"hermes-agent/hermes_cli/default_soul.py",
		"internal/hermes/default_soul.go",
		"docs/content/building-gormes/architecture_plan/hermes-source-pairs.json",
	} {
		abs := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
	}
	manifest := `{
  "schema_version": "1.0",
  "hermes_sha": "abc123",
  "pairs": [
    {
      "hermes_file": "hermes_cli/default_soul.py",
      "gormes_targets": ["internal/hermes/default_soul.go"],
      "status": "covered",
      "contract": "Default SOUL.md seed text",
      "tests": ["go test ./internal/hermes -run TestDefaultSoul -count=1"],
      "last_checked_hermes_sha": "abc123"
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(root, "docs/content/building-gormes/architecture_plan/hermes-source-pairs.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "hermes-agent/hermes_cli/default_soul.py"), []byte("fixture\n"), 0o644); err != nil {
		t.Fatalf("write hermes file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal/hermes/default_soul.go"), []byte("fixture\n"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	var stdout, stderr bytes.Buffer
	err := run(&stdout, &stderr, []string{"--repo-root", root, "hermes-source-pairs", "sync-sha", "--allow-unmapped-high-priority", "--hermes-sha", "def456"})
	if err != nil {
		t.Fatalf("run sync-sha error = %v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "repoctl: hermes-source-pairs sync-sha ok pairs=1 hermes_sha=def456") {
		t.Fatalf("stdout = %q, want sync summary", stdout.String())
	}
	raw, err := os.ReadFile(filepath.Join(root, "docs/content/building-gormes/architecture_plan/hermes-source-pairs.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if !strings.Contains(string(raw), `"hermes_sha": "def456"`) || !strings.Contains(string(raw), `"last_checked_hermes_sha": "def456"`) {
		t.Fatalf("manifest was not synced:\n%s", raw)
	}
}

package repoctl

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/progress"
	"github.com/TrebuchetDynamics/gormes-agent/internal/progress/fidelity"
)

func TestWriteHermesContractInventoryWritesJSONAndMarkdown(t *testing.T) {
	root := hermesContractInventoryFixtureRoot(t)

	result, err := WriteHermesContractInventory(HermesContractInventoryOptions{
		Root:             root,
		CurrentHermesSHA: "abc123",
	})
	if err != nil {
		t.Fatalf("WriteHermesContractInventory: %v", err)
	}
	raw, err := os.ReadFile(result.JSONPath)
	if err != nil {
		t.Fatalf("read json report: %v", err)
	}
	var report fidelity.Report
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode json report: %v\n%s", err, raw)
	}
	if report.SchemaVersion != fidelity.SchemaVersion || report.HermesSHA != "abc123" || report.Summary.Total != 10 {
		t.Fatalf("report identity = schema %q sha %q total %d", report.SchemaVersion, report.HermesSHA, report.Summary.Total)
	}
	if report.Candidates.CLI == nil || report.Candidates.Tools == nil || report.Candidates.Memory == nil {
		t.Fatalf("candidate inventories must be present: %+v", report.Candidates)
	}
	if len(report.ReleaseCheckpoints) == 0 {
		t.Fatalf("release checkpoints must be present")
	}
	if len(report.ContinuityCategories) < 14 {
		t.Fatalf("continuity categories = %d, want at least 14", len(report.ContinuityCategories))
	}
	if len(report.UnmappedUpstream.SourceFiles) == 0 || len(report.UnmappedUpstream.DocsFiles) == 0 || len(report.UnmappedUpstream.TestFiles) == 0 {
		t.Fatalf("unmapped upstream evidence must include source/docs/tests: %+v", report.UnmappedUpstream)
	}
	if suite := findUnmappedTestSuite(report, "hermes_cli"); suite.Count != 1 || suite.SourcePrefix != "hermes_cli" || len(suite.ProgressRows) == 0 {
		t.Fatalf("hermes_cli unmapped test suite = %+v, want count/source/progress evidence", suite)
	}
	if got := findInventorySurface(report, "goncho_memory").Status; got != fidelity.StatusCovered {
		t.Fatalf("goncho_memory status = %q, want covered", got)
	}

	md, err := os.ReadFile(result.MarkdownPath)
	if err != nil {
		t.Fatalf("read markdown report: %v", err)
	}
	for _, want := range []string{
		"# Hermes Contract Inventory",
		"progress.json` remains the only backlog",
		"Gormes may claim all Hermes features and architecture are paired",
		"| `goncho_memory` | `covered` | `none` |",
		"## Release Checkpoints",
		"## Per-Module Gap Summary",
		"## Continuity Categories",
		"## Unmapped Upstream Evidence",
		"## Unmapped Test Suite Classification",
		"| `hermes_cli` | `1` | `hermes_cli` |",
		"## Candidate Inventory",
	} {
		if !strings.Contains(string(md), want) {
			t.Fatalf("markdown report missing %q:\n%s", want, md)
		}
	}
}

func TestWriteHermesContractInventoryPreservesTimestampWhenContentIsUnchanged(t *testing.T) {
	root := hermesContractInventoryFixtureRoot(t)
	first := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)
	second := first.Add(2 * time.Hour)

	result, err := WriteHermesContractInventory(HermesContractInventoryOptions{
		Root:             root,
		CurrentHermesSHA: "abc123",
		Now: func() time.Time {
			return first
		},
	})
	if err != nil {
		t.Fatalf("first WriteHermesContractInventory: %v", err)
	}
	firstJSON := readFileForInventoryTest(t, result.JSONPath)
	firstMD := readFileForInventoryTest(t, result.MarkdownPath)

	result, err = WriteHermesContractInventory(HermesContractInventoryOptions{
		Root:             root,
		CurrentHermesSHA: "abc123",
		Now: func() time.Time {
			return second
		},
	})
	if err != nil {
		t.Fatalf("second WriteHermesContractInventory: %v", err)
	}
	secondJSON := readFileForInventoryTest(t, result.JSONPath)
	secondMD := readFileForInventoryTest(t, result.MarkdownPath)
	if string(firstJSON) != string(secondJSON) {
		t.Fatalf("JSON report changed on no-op rewrite:\nfirst:\n%s\nsecond:\n%s", firstJSON, secondJSON)
	}
	if string(firstMD) != string(secondMD) {
		t.Fatalf("Markdown report changed on no-op rewrite:\nfirst:\n%s\nsecond:\n%s", firstMD, secondMD)
	}
	if result.Report.GeneratedAt != first.UTC().Format(time.RFC3339) {
		t.Fatalf("GeneratedAt = %q, want preserved %q", result.Report.GeneratedAt, first.UTC().Format(time.RFC3339))
	}
}

func readFileForInventoryTest(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}

func findUnmappedTestSuite(report fidelity.Report, suite string) fidelity.UpstreamUnmappedTestSuite {
	for _, row := range report.UnmappedUpstream.TestSuites {
		if row.Suite == suite {
			return row
		}
	}
	return fidelity.UpstreamUnmappedTestSuite{}
}

func findInventorySurface(report fidelity.Report, id string) fidelity.SurfaceReport {
	for _, surface := range report.Surfaces {
		if surface.ID == id {
			return surface
		}
	}
	return fidelity.SurfaceReport{}
}

func hermesContractInventoryFixtureRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	hermes := filepath.Join(root, "hermes-agent")
	for _, rel := range []string{
		"hermes_cli/main.py",
		"hermes_cli/profiles.py",
		"hermes_cli/session_recap.py",
		"run_agent.py",
		"gateway/session.py",
		"tools/session_search_tool.py",
		"tools/memory_tool.py",
		"agent/memory_manager.py",
		"plugins/memory/__init__.py",
		"agent/curator.py",
		"hermes_cli/curator.py",
		"tools/skill_usage.py",
		"agent/prompt_builder.py",
		"agent/skill_commands.py",
		"agent/skill_preprocessing.py",
		"agent/skill_utils.py",
		"hermes_cli/auth_commands.py",
		"hermes_cli/providers.py",
		"hermes_cli/setup.py",
		"gateway/run.py",
		"tools/registry.py",
		"tools/file_tools.py",
		"tools/code_execution_tool.py",
		"hermes_cli/mcp_config.py",
		"tools/mcp_tool.py",
		"acp_adapter/entry.py",
		"acp_adapter/server.py",
		"ui-tui/package.json",
		"cli.py",
		"RELEASE_v0.14.0.md",
		"docs/unmapped.md",
		"tools/unmapped_tool.py",
		"tests/hermes_cli/test_unmapped.py",
	} {
		path := filepath.Join(hermes, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte("# fixture\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	writeInventorySourcePairs(t, root)
	writeInventoryProgress(t, root)
	return root
}

func writeInventorySourcePairs(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, "webpages", "docs", "content", "building-gormes", "architecture_plan", "hermes-source-pairs.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir source-pairs: %v", err)
	}
	body := `{
  "schema_version": "1.0",
  "hermes_sha": "abc123",
  "pairs": [
    {
      "hermes_file": "tools/memory_tool.py",
      "gormes_targets": ["internal/goncho/memory.go"],
      "status": "covered",
      "contract": "Memory and Goncho recall contract.",
      "tests": ["go test ./internal/goncho -count=1"],
      "progress_rows": ["Goncho memory parity"],
      "last_checked_hermes_sha": "abc123"
    }
  ]
}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write source-pairs: %v", err)
	}
}

func writeInventoryProgress(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, "webpages", "docs", "content", "building-gormes", "architecture_plan", "progress.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir progress: %v", err)
	}
	p := &progress.Progress{
		Meta: progress.Meta{Version: "test"},
		Phases: map[string]progress.Phase{
			"1": {
				Name:        "Fidelity",
				Deliverable: "Fixture",
				Subphases: map[string]progress.Subphase{
					"1.A": {
						Name: "Continuity",
						Items: []progress.Item{
							{
								Name:           "Goncho memory parity",
								Priority:       "P0",
								Status:         progress.StatusComplete,
								ContractStatus: progress.ContractStatusValidated,
								Module:         "goncho",
								Contract:       "Goncho memory recall mirrors Hermes memory behavior.",
								SourceRefs:     []string{"hermes-agent/tools/memory_tool.py", "internal/goncho/memory.go"},
								TestCommands:   []string{"go test ./internal/goncho -count=1"},
							},
							{
								Name:           "Provider auth setup parity",
								Priority:       "P0",
								Status:         progress.StatusPlanned,
								ContractStatus: progress.ContractStatusDraft,
								Module:         "providers",
								Contract:       "Provider auth setup must mirror Hermes auth commands.",
								SourceRefs:     []string{"hermes-agent/hermes_cli/auth_commands.py", "cmd/gormes/auth.go"},
								TestCommands:   []string{"go test ./cmd/gormes -run TestAuth -count=1"},
							},
						},
					},
				},
			},
		},
	}
	if err := progress.SaveProgress(path, p); err != nil {
		t.Fatalf("write progress: %v", err)
	}
}

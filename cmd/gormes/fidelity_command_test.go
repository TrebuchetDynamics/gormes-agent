package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/progress"
)

func TestFidelityHermesCommandJSONReportOnly(t *testing.T) {
	root := writeFidelityCommandFixture(t)

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}),
		"fidelity", "hermes",
		"--repo-root", root,
		"--progress", filepath.Join(root, "webpages/docs/content/building-gormes/architecture_plan/progress.json"),
		"--hermes", filepath.Join(root, "hermes-agent"),
		"--hermes-sha", "abc123",
		"--json",
	)
	if err != nil {
		t.Fatalf("fidelity hermes --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if strings.Contains(stdout, ".gormes") || strings.Contains(stdout, ".hermes") {
		t.Fatalf("report leaked live home paths:\n%s", stdout)
	}

	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Action string `json:"action"`
		OK     bool   `json:"ok"`
		Report struct {
			HermesSHA string `json:"hermes_sha"`
			Summary   struct {
				Total int `json:"total"`
			} `json:"summary"`
			Surfaces []struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"surfaces"`
		} `json:"report"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not JSON: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if got.Build.Version != Version || got.Action != "fidelity_hermes" || !got.OK || got.Report.HermesSHA != "abc123" || got.Report.Summary.Total != 10 {
		t.Fatalf("unexpected report header: %+v", got)
	}
	if !commandReportHasSurface(got.Report.Surfaces, "goncho_memory", "covered") {
		t.Fatalf("report missing covered goncho_memory surface: %+v", got.Report.Surfaces)
	}
}

func TestFidelityHermesCommandStrictReturnsNonZeroAfterEmittingJSON(t *testing.T) {
	root := writeFidelityCommandFixture(t)

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}),
		"fidelity", "hermes",
		"--repo-root", root,
		"--progress", filepath.Join(root, "webpages/docs/content/building-gormes/architecture_plan/progress.json"),
		"--hermes", filepath.Join(root, "hermes-agent"),
		"--hermes-sha", "abc123",
		"--strict",
		"--json",
	)
	if err == nil {
		t.Fatalf("strict fidelity report err=nil, want non-zero because fixture has gaps\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if code := exitCodeFromError(err); code != 1 {
		t.Fatalf("strict exit code = %d, want 1; err=%v\nstdout=%s\nstderr=%s", code, err, stdout, stderr)
	}
	var got struct {
		Action string `json:"action"`
		OK     bool   `json:"ok"`
		Report struct {
			Strict bool `json:"strict"`
		} `json:"report"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("strict stdout is not JSON: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if got.Action != "fidelity_hermes" || got.OK || !got.Report.Strict {
		t.Fatalf("strict report = %+v, want action with ok=false strict=true", got)
	}
}

func commandReportHasSurface(surfaces []struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}, id, status string) bool {
	for _, surface := range surfaces {
		if surface.ID == id && surface.Status == status {
			return true
		}
	}
	return false
}

func writeFidelityCommandFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	hermes := filepath.Join(root, "hermes-agent")
	for _, rel := range []string{
		"hermes_cli/profiles.py",
		"hermes_cli/main.py",
		"hermes_cli/session_recap.py",
		"agent/prompt_builder.py",
		"agent/skill_commands.py",
		"agent/skill_preprocessing.py",
		"agent/skill_utils.py",
		"run_agent.py",
		"gateway/run.py",
		"gateway/session.py",
		"tools/memory_tool.py",
		"agent/memory_manager.py",
		"plugins/memory/__init__.py",
		"agent/curator.py",
		"hermes_cli/curator.py",
		"tools/session_search_tool.py",
		"tools/skill_usage.py",
		"tools/skills_tool.py",
		"tools/skill_manager_tool.py",
		"tools/skills_sync.py",
		"tools/registry.py",
		"tools/file_tools.py",
		"tools/code_execution_tool.py",
		"hermes_cli/auth_commands.py",
		"hermes_cli/providers.py",
		"hermes_cli/setup.py",
		"hermes_cli/send_cmd.py",
		"hermes_cli/mcp_config.py",
		"tools/mcp_tool.py",
		"acp_adapter/entry.py",
		"acp_adapter/server.py",
		"ui-tui/package.json",
		"cli.py",
	} {
		path := filepath.Join(hermes, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte("# fixture\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	sourcePairs := filepath.Join(root, "webpages", "docs", "content", "building-gormes", "architecture_plan", "hermes-source-pairs.json")
	if err := os.MkdirAll(filepath.Dir(sourcePairs), 0o755); err != nil {
		t.Fatalf("mkdir source-pairs: %v", err)
	}
	if err := os.WriteFile(sourcePairs, []byte(`{
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
}`), 0o644); err != nil {
		t.Fatalf("write source-pairs: %v", err)
	}

	progressPath := filepath.Join(root, "webpages", "docs", "content", "building-gormes", "architecture_plan", "progress.json")
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
						},
					},
				},
			},
		},
	}
	if err := progress.SaveProgress(progressPath, p); err != nil {
		t.Fatalf("write progress: %v", err)
	}
	return root
}

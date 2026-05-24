package fidelity

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/progress"
)

func TestHermesReportClassifiesCriticalSurfacesFromStaticEvidence(t *testing.T) {
	root := t.TempDir()
	hermes := filepath.Join(root, "hermes-agent")
	writeHermesSurfaceFixtures(t, hermes)
	writeSourcePairsFixture(t, root, "abc123")
	progressPath := writeProgressFixture(t, root)

	report, err := GenerateHermesReport(context.Background(), Options{
		RepoRoot:     root,
		ProgressPath: progressPath,
		HermesPath:   hermes,
		GitRevision: func(context.Context, string) (string, error) {
			return "abc123", nil
		},
	})
	if err != nil {
		t.Fatalf("GenerateHermesReport: %v", err)
	}

	if report.SchemaVersion == "" || report.HermesSHA != "abc123" || len(report.Surfaces) != 10 {
		t.Fatalf("report identity/surfaces = schema %q sha %q surfaces %d", report.SchemaVersion, report.HermesSHA, len(report.Surfaces))
	}
	if report.Summary.Total != 10 || report.Summary.Critical != 10 {
		t.Fatalf("summary = %+v, want 10 total critical surfaces", report.Summary)
	}
	if report.Summary.UnmappedUpstreamFiles == 0 || len(report.UnmappedUpstream.SourceFiles) == 0 || len(report.UnmappedUpstream.DocsFiles) == 0 || len(report.UnmappedUpstream.TestFiles) == 0 {
		t.Fatalf("unmapped upstream evidence missing from report: summary=%+v unmapped=%+v", report.Summary, report.UnmappedUpstream)
	}
	if suite := unmappedTestSuiteByID(report, "hermes_cli"); suite.Count != 1 || suite.SourcePrefix != "hermes_cli" || len(suite.Examples) != 1 || len(suite.ProgressRows) == 0 {
		t.Fatalf("hermes_cli unmapped test suite = %+v, want actionable suite/source/progress grouping", suite)
	}
	if len(report.ReleaseCheckpoints) == 0 {
		t.Fatalf("release checkpoints missing from report")
	}
	if len(report.ContinuityCategories) < 14 {
		t.Fatalf("continuity categories = %d, want at least 14", len(report.ContinuityCategories))
	}

	goncho := surfaceByID(t, report, "goncho_memory")
	if goncho.Status != StatusCovered {
		t.Fatalf("goncho_memory status = %q, want covered; surface=%+v", goncho.Status, goncho)
	}
	if len(goncho.ProgressRows) == 0 || len(goncho.SourcePairs) == 0 || len(goncho.UpstreamRefs) == 0 || len(goncho.GormesRefs) == 0 {
		t.Fatalf("goncho_memory missing joined evidence: %+v", goncho)
	}

	provider := surfaceByID(t, report, "provider_auth_setup")
	if provider.Status != StatusPartial {
		t.Fatalf("provider_auth_setup status = %q, want partial from source-pair evidence; surface=%+v", provider.Status, provider)
	}

	learning := surfaceByID(t, report, "learning_loop")
	if learning.Status != StatusPlanned {
		t.Fatalf("learning_loop status = %q, want planned from progress-only row; surface=%+v", learning.Status, learning)
	}

	mcp := surfaceByID(t, report, "mcp_acp")
	if mcp.Status != StatusMissing {
		t.Fatalf("mcp_acp status = %q, want missing when no rows or source pairs map it; surface=%+v", mcp.Status, mcp)
	}

	if !report.OK {
		t.Fatalf("report-only mode must stay OK even with gaps: %+v", report.Summary)
	}
}

func TestHermesReportStrictModeFailsOnUncoveredCriticalSurface(t *testing.T) {
	root := t.TempDir()
	hermes := filepath.Join(root, "hermes-agent")
	writeHermesSurfaceFixtures(t, hermes)
	writeSourcePairsFixture(t, root, "abc123")
	progressPath := writeProgressFixture(t, root)

	report, err := GenerateHermesReport(context.Background(), Options{
		RepoRoot:     root,
		ProgressPath: progressPath,
		HermesPath:   hermes,
		Strict:       true,
		GitRevision: func(context.Context, string) (string, error) {
			return "abc123", nil
		},
	})
	if err != nil {
		t.Fatalf("GenerateHermesReport: %v", err)
	}
	if report.OK {
		t.Fatalf("strict report OK=true with uncovered surfaces: %+v", report.Summary)
	}
}

func unmappedTestSuiteByID(report Report, id string) UpstreamUnmappedTestSuite {
	for _, suite := range report.UnmappedUpstream.TestSuites {
		if suite.Suite == id {
			return suite
		}
	}
	return UpstreamUnmappedTestSuite{}
}

func surfaceByID(t *testing.T, report Report, id string) SurfaceReport {
	t.Helper()
	for _, surface := range report.Surfaces {
		if surface.ID == id {
			return surface
		}
	}
	t.Fatalf("surface %q missing from report: %+v", id, report.Surfaces)
	return SurfaceReport{}
}

func writeHermesSurfaceFixtures(t *testing.T, hermes string) {
	t.Helper()
	for _, surface := range defaultSurfaces() {
		for _, ref := range surface.UpstreamRefs {
			path := filepath.Join(hermes, filepath.FromSlash(ref))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatalf("mkdir %s: %v", path, err)
			}
			if err := os.WriteFile(path, []byte("# fixture\n"), 0o644); err != nil {
				t.Fatalf("write %s: %v", path, err)
			}
		}
	}
	for _, rel := range []string{
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
}

func writeSourcePairsFixture(t *testing.T, root, sha string) {
	t.Helper()
	path := filepath.Join(root, "webpages", "docs", "content", "building-gormes", "architecture_plan", "hermes-source-pairs.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir source-pairs: %v", err)
	}
	body := `{
  "schema_version": "1.0",
  "hermes_sha": "` + sha + `",
  "pairs": [
    {
      "hermes_file": "tools/memory_tool.py",
      "gormes_targets": ["internal/goncho/memory.go"],
      "status": "covered",
      "contract": "Memory and Goncho recall contract.",
      "tests": ["go test ./internal/goncho -count=1"],
      "progress_rows": ["Goncho memory parity"],
      "last_checked_hermes_sha": "` + sha + `"
    },
    {
      "hermes_file": "hermes_cli/auth_commands.py",
      "gormes_targets": ["cmd/gormes/auth.go"],
      "status": "partial",
      "contract": "Auth and provider setup contract.",
      "tests": ["go test ./cmd/gormes -run TestAuth -count=1"],
      "progress_rows": ["Provider auth setup parity"],
      "last_checked_hermes_sha": "` + sha + `"
    }
  ]
}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write source-pairs: %v", err)
	}
}

func writeProgressFixture(t *testing.T, root string) string {
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
							{
								Name:           "Learning-loop curator behavior",
								Priority:       "P0",
								Status:         progress.StatusPlanned,
								ContractStatus: progress.ContractStatusDraft,
								Module:         "skills",
								Contract:       "Curator and candidate learning-loop updates remain auditable.",
								SourceRefs:     []string{"hermes-agent/agent/curator.py", "cmd/gormes/curator.go"},
								TestCommands:   []string{"go test ./cmd/gormes -run TestCurator -count=1"},
							},
						},
					},
				},
			},
		},
	}
	if err := progress.SaveProgress(path, p); err != nil {
		t.Fatalf("write progress fixture: %v", err)
	}
	return path
}

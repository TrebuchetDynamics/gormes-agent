package repoctl

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSourcePairsRejectsCoveredWithoutTests(t *testing.T) {
	root := sourcePairFixtureRoot(t)
	writeSourcePairsManifestText(t, root, `{
  "schema_version": "1.0",
  "hermes_sha": "abcdef123456",
  "pairs": [
    {
      "hermes_file": "hermes_cli/default_soul.py",
      "gormes_targets": ["internal/hermes/default_soul.go"],
      "status": "covered",
      "contract": "Default SOUL.md seed text",
      "last_checked_hermes_sha": "abcdef123456"
    }
  ]
}`)

	_, err := ValidateSourcePairs(SourcePairOptions{Root: root, RequireHighPriority: false})
	if err == nil || !strings.Contains(err.Error(), "covered row requires tests") {
		t.Fatalf("ValidateSourcePairs error = %v, want covered tests error", err)
	}
}

func TestValidateSourcePairsRejectsMissingTarget(t *testing.T) {
	root := sourcePairFixtureRoot(t)
	writeSourcePairsManifestText(t, root, `{
  "schema_version": "1.0",
  "hermes_sha": "abcdef123456",
  "pairs": [
    {
      "hermes_file": "hermes_cli/default_soul.py",
      "gormes_targets": ["internal/hermes/missing.go"],
      "status": "covered",
      "contract": "Default SOUL.md seed text",
      "tests": ["go test ./internal/hermes -run TestDefaultSoul -count=1"],
      "last_checked_hermes_sha": "abcdef123456"
    }
  ]
}`)

	_, err := ValidateSourcePairs(SourcePairOptions{Root: root, RequireHighPriority: false})
	if err == nil || !strings.Contains(err.Error(), "missing Gormes target") {
		t.Fatalf("ValidateSourcePairs error = %v, want missing target error", err)
	}
}

func TestValidateSourcePairsRejectsStaleHermesSHA(t *testing.T) {
	root := sourcePairFixtureRoot(t)
	writeSourcePairsManifestText(t, root, `{
  "schema_version": "1.0",
  "hermes_sha": "deadbeef",
  "pairs": [
    {
      "hermes_file": "hermes_cli/default_soul.py",
      "gormes_targets": ["internal/hermes/default_soul.go"],
      "status": "covered",
      "contract": "Default SOUL.md seed text",
      "tests": ["go test ./internal/hermes -run TestDefaultSoul -count=1"],
      "last_checked_hermes_sha": "deadbeef"
    }
  ]
}`)

	_, err := ValidateSourcePairs(SourcePairOptions{
		Root:                root,
		RequireHighPriority: false,
		CurrentHermesSHA:    "abcdef123456",
	})
	if err == nil || !strings.Contains(err.Error(), "stale Hermes SHA") {
		t.Fatalf("ValidateSourcePairs error = %v, want stale SHA error", err)
	}
}

func TestValidateSourcePairsRejectsUnmappedHighPriorityFile(t *testing.T) {
	root := sourcePairFixtureRoot(t)
	writeSourcePairsManifestText(t, root, `{
  "schema_version": "1.0",
  "hermes_sha": "abcdef123456",
  "pairs": [
    {
      "hermes_file": "hermes_cli/default_soul.py",
      "gormes_targets": ["internal/hermes/default_soul.go"],
      "status": "covered",
      "contract": "Default SOUL.md seed text",
      "tests": ["go test ./internal/hermes -run TestDefaultSoul -count=1"],
      "last_checked_hermes_sha": "abcdef123456"
    }
  ]
}`)

	_, err := ValidateSourcePairs(SourcePairOptions{Root: root, RequireHighPriority: true})
	if err == nil || !strings.Contains(err.Error(), "high-priority Hermes file is unmapped") {
		t.Fatalf("ValidateSourcePairs error = %v, want high-priority unmapped error", err)
	}
}

func TestRenderSourcePairsReportSummarizesCounts(t *testing.T) {
	root := sourcePairFixtureRoot(t)
	writeSourcePairsManifestText(t, root, `{
  "schema_version": "1.0",
  "hermes_sha": "abcdef123456",
  "pairs": [
    {
      "hermes_file": "hermes_cli/default_soul.py",
      "gormes_targets": ["internal/hermes/default_soul.go"],
      "status": "covered",
      "contract": "Default SOUL.md seed text",
      "tests": ["go test ./internal/hermes -run TestDefaultSoul -count=1"],
      "progress_rows": ["Default agent identity / SOUL.md loader"],
      "upstream_tests": ["tests/hermes_cli/test_config.py"],
      "last_checked_hermes_sha": "abcdef123456",
      "notes": "Gormes identity substitution is intentional."
    },
    {
      "hermes_file": "hermes_cli/config.py",
      "gormes_targets": ["internal/config"],
      "status": "partial",
      "contract": "Config loading and default home setup",
      "tests": ["go test ./internal/config -count=1"],
      "progress_rows": ["Gormes config command surface"],
      "last_checked_hermes_sha": "abcdef123456"
    }
  ]
}`)

	report, err := RenderSourcePairsReport(SourcePairOptions{Root: root, RequireHighPriority: false})
	if err != nil {
		t.Fatalf("RenderSourcePairsReport: %v", err)
	}
	for _, want := range []string{
		"# Hermes Source Pairs",
		"- `covered`: 1",
		"- `partial`: 1",
		"| `hermes_cli/default_soul.py` | `covered` |",
		"`internal/hermes/default_soul.go`",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing %q:\n%s", want, report)
		}
	}
}

func TestSyncSourcePairsSHAUpdatesManifestRows(t *testing.T) {
	root := sourcePairFixtureRoot(t)
	writeSourcePairsManifestText(t, root, `{
  "schema_version": "1.0",
  "hermes_sha": "abcdef123456",
  "pairs": [
    {
      "hermes_file": "hermes_cli/default_soul.py",
      "gormes_targets": ["internal/hermes/default_soul.go"],
      "status": "covered",
      "contract": "Default SOUL.md seed text",
      "tests": ["go test ./internal/hermes -run TestDefaultSoul -count=1"],
      "last_checked_hermes_sha": "abcdef123456"
    }
  ]
}`)

	result, err := SyncSourcePairsSHA(SourcePairOptions{
		Root:                root,
		CurrentHermesSHA:    "feedface987654",
		RequireHighPriority: false,
	})
	if err != nil {
		t.Fatalf("SyncSourcePairsSHA: %v", err)
	}
	if result.Manifest.HermesSHA != "feedface987654" {
		t.Fatalf("manifest hermes_sha = %q, want updated SHA", result.Manifest.HermesSHA)
	}
	if got := result.Manifest.Pairs[0].LastCheckedHermesSHA; got != "feedface987654" {
		t.Fatalf("row last_checked_hermes_sha = %q, want updated SHA", got)
	}
	if len(result.DemotedCovered) != 0 {
		t.Fatalf("DemotedCovered = %v, want none when changed files are unknown", result.DemotedCovered)
	}
}

func TestSyncSourcePairsSHADemotesCoveredChangedSource(t *testing.T) {
	root := sourcePairFixtureRoot(t)
	oldSHA, newSHA := initSourcePairGitHistory(t, filepath.Join(root, "hermes-agent"), "hermes_cli/default_soul.py")
	writeSourcePairsManifestText(t, root, `{
  "schema_version": "1.0",
  "hermes_sha": "`+oldSHA+`",
  "pairs": [
    {
      "hermes_file": "hermes_cli/default_soul.py",
      "gormes_targets": ["internal/hermes/default_soul.go"],
      "status": "covered",
      "contract": "Default SOUL.md seed text",
      "tests": ["go test ./internal/hermes -run TestDefaultSoul -count=1"],
      "last_checked_hermes_sha": "`+oldSHA+`"
    }
  ]
}`)

	result, err := SyncSourcePairsSHA(SourcePairOptions{
		Root:                root,
		CurrentHermesSHA:    newSHA,
		RequireHighPriority: false,
	})
	if err != nil {
		t.Fatalf("SyncSourcePairsSHA: %v", err)
	}
	if got := result.Manifest.Pairs[0].Status; got != "partial" {
		t.Fatalf("status = %q, want covered row demoted to partial after source changed", got)
	}
	if len(result.DemotedCovered) != 1 || result.DemotedCovered[0] != "hermes_cli/default_soul.py" {
		t.Fatalf("DemotedCovered = %v, want default_soul demotion", result.DemotedCovered)
	}
	if !strings.Contains(result.Manifest.Pairs[0].Notes, "upstream source changed") {
		t.Fatalf("notes = %q, want upstream-change review note", result.Manifest.Pairs[0].Notes)
	}
}

func sourcePairFixtureRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, path := range []string{
		"hermes-agent/hermes_cli/default_soul.py",
		"hermes-agent/hermes_cli/config.py",
		"hermes-agent/tests/hermes_cli/test_config.py",
		"internal/hermes/default_soul.go",
		"internal/config/placeholder.go",
	} {
		abs := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(abs, []byte("fixture\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	return root
}

func writeSourcePairsManifestText(t *testing.T, root, body string) {
	t.Helper()
	path := filepath.Join(root, sourcePairsManifestRel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir manifest dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

func initSourcePairGitHistory(t *testing.T, repo, rel string) (string, string) {
	t.Helper()
	runSourcePairGit(t, repo, "init")
	runSourcePairGit(t, repo, "add", rel)
	runSourcePairGit(t, repo, "-c", "user.name=Gormes Test", "-c", "user.email=gormes@example.test", "commit", "-m", "initial")
	oldSHA := strings.TrimSpace(runSourcePairGit(t, repo, "rev-parse", "HEAD"))
	if err := os.WriteFile(filepath.Join(repo, filepath.FromSlash(rel)), []byte("changed\n"), 0o644); err != nil {
		t.Fatalf("write changed source: %v", err)
	}
	runSourcePairGit(t, repo, "add", rel)
	runSourcePairGit(t, repo, "-c", "user.name=Gormes Test", "-c", "user.email=gormes@example.test", "commit", "-m", "change source")
	newSHA := strings.TrimSpace(runSourcePairGit(t, repo, "rev-parse", "HEAD"))
	return oldSHA, newSHA
}

func runSourcePairGit(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

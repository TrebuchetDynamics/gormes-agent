package docs_test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type deltaEntry struct {
	Path     string `json:"path"`
	Status   string `json:"status"`
	Category string `json:"category"`
	Family   string `json:"family"`
}

type deltaReport struct {
	SchemaVersion int            `json:"schema_version"`
	BaselineSHA   string         `json:"baseline_sha"`
	TargetSHA     string         `json:"target_sha"`
	CommitCount   int            `json:"commit_count"`
	TotalPaths    int            `json:"total_paths"`
	ByCategory    map[string]int `json:"by_category"`
	Entries       []deltaEntry   `json:"entries"`
}

const (
	deltaBaselineSHA = "d0288be5b3330d2442e3907185b8e9d0958297bb"
	deltaTargetSHA   = "197f16a12b098f1ea541bbd0a5ddd7d66f2d2dee"
)

func TestUpstreamDeltaCoversEveryChangedPath(t *testing.T) {
	root := findRepoRoot(t)
	hermesDir := filepath.Join(root, "hermes-agent")
	for _, sha := range []string{deltaBaselineSHA, deltaTargetSHA} {
		if err := exec.Command("git", "-C", hermesDir, "cat-file", "-e", sha+"^{commit}").Run(); err != nil {
			t.Skipf("upstream commit %s not present in %s", sha, hermesDir)
		}
	}

	reportPath := filepath.Join(root, "webpages", "docs", "parity-evidence",
		fmt.Sprintf("UPSTREAM-DELTA-%s-%s.json", deltaBaselineSHA[:12], deltaTargetSHA[:12]))
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read delta report: %v", err)
	}
	var report deltaReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode delta report: %v", err)
	}
	if report.BaselineSHA != deltaBaselineSHA || report.TargetSHA != deltaTargetSHA {
		t.Fatalf("report SHAs = %s..%s", report.BaselineSHA, report.TargetSHA)
	}

	countOut, err := exec.Command("git", "-C", hermesDir, "rev-list", "--count", deltaBaselineSHA+".."+deltaTargetSHA).Output()
	if err != nil {
		t.Fatalf("count commits: %v", err)
	}
	var commits int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(countOut)), "%d", &commits); err != nil {
		t.Fatalf("parse commit count %q: %v", countOut, err)
	}
	if report.CommitCount != commits {
		t.Fatalf("report commits = %d, want %d", report.CommitCount, commits)
	}

	diffOut, err := exec.Command("git", "-C", hermesDir, "diff", "--name-status", "--no-renames",
		deltaBaselineSHA+".."+deltaTargetSHA, "--").Output()
	if err != nil {
		t.Fatalf("list changed paths: %v", err)
	}
	diff := map[string]string{}
	scanner := bufio.NewScanner(strings.NewReader(string(diffOut)))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		fields := strings.Split(strings.TrimSpace(scanner.Text()), "\t")
		if len(fields) < 2 {
			continue
		}
		diff[filepath.ToSlash(fields[1])] = strings.TrimSpace(fields[0])
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan diff: %v", err)
	}

	seen := map[string]deltaEntry{}
	allowed := map[string]bool{"runtime": true, "app-ui": true, "test": true, "docs": true, "tooling-exclusion": true}
	byCategory := map[string]int{}
	for _, entry := range report.Entries {
		if _, exists := seen[entry.Path]; exists {
			t.Fatalf("duplicate delta entry %q", entry.Path)
		}
		seen[entry.Path] = entry
		if !allowed[entry.Category] {
			t.Fatalf("path %q has unsupported category %q", entry.Path, entry.Category)
		}
		byCategory[entry.Category]++
	}
	if len(seen) != len(diff) {
		t.Fatalf("delta entries = %d, changed paths = %d", len(seen), len(diff))
	}
	for path, status := range diff {
		entry, ok := seen[path]
		if !ok {
			t.Fatalf("changed path missing from delta report: %s", path)
		}
		if entry.Status != status {
			t.Fatalf("path %q status = %q, want %q", path, entry.Status, status)
		}
	}
	for path := range seen {
		if _, ok := diff[path]; !ok {
			t.Fatalf("delta report contains path not in git diff: %s", path)
		}
	}
	if report.TotalPaths != len(diff) {
		t.Fatalf("report total = %d, want %d", report.TotalPaths, len(diff))
	}
	for category, want := range report.ByCategory {
		if got := byCategory[category]; got != want {
			t.Fatalf("category %q count = %d, want %d", category, got, want)
		}
	}
	if byCategory["other"] != 0 {
		t.Fatalf("unclassified paths remain: %d", byCategory["other"])
	}
}

func TestUpstreamDeltaMarkdownSummarizesReport(t *testing.T) {
	root := findRepoRoot(t)
	path := filepath.Join(root, "webpages", "docs", "parity-evidence",
		fmt.Sprintf("UPSTREAM-DELTA-%s-%s.md", deltaBaselineSHA[:12], deltaTargetSHA[:12]))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read delta markdown: %v", err)
	}
	body := string(raw)
	for _, want := range []string{
		"Baseline SHA: `" + deltaBaselineSHA + "`",
		"Target SHA: `" + deltaTargetSHA + "`",
		"Commits: **17230**",
		"Changed paths: **13572**",
		"| `runtime` | 3669 |",
		"| `app-ui` | 1999 |",
		"| `test` | 6552 |",
		"| `docs` | 1098 |",
		"| `tooling-exclusion` | 254 |",
		"## Runtime Feature Families",
		"## App / UI Families",
		"## Method",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("delta markdown missing %q", want)
		}
	}
}

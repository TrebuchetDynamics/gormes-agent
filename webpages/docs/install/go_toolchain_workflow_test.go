package install_test

import (
	"strings"
	"testing"
)

func TestGitHubActionsUseGoModToolchain(t *testing.T) {
	goMod := readRepoFile(t, "go.mod")
	if !strings.Contains(goMod, "\ngo 1.26.4\n") {
		t.Fatalf("go.mod must declare the current patched Go floor; got:\n%s", goMod)
	}

	workflows := map[string]int{
		".github/workflows/ci.yml":                2,
		".github/workflows/deploy-gormes-www.yml": 1,
		".github/workflows/release-prep.yml":      1,
		".github/workflows/release.yml":           2,
	}
	for rel, setupCount := range workflows {
		rel := rel
		setupCount := setupCount
		t.Run(rel, func(t *testing.T) {
			workflow := readRepoFile(t, rel)
			if got := strings.Count(workflow, "uses: actions/setup-go@v6"); got != setupCount {
				t.Fatalf("%s setup-go action count = %d, want %d with current setup-go docs", rel, got, setupCount)
			}
			if got := strings.Count(workflow, "go-version-file: go.mod"); got != setupCount {
				t.Fatalf("%s go-version-file count = %d, want %d", rel, got, setupCount)
			}
			if strings.Contains(workflow, "go-version: '1.25'") || strings.Contains(workflow, "go-version: \"1.25\"") {
				t.Fatalf("%s still hard-codes stale Go 1.25 instead of reading go.mod", rel)
			}
			if strings.Contains(workflow, "go-version:") {
				t.Fatalf("%s must not set go-version because setup-go gives that precedence over go-version-file", rel)
			}
		})
	}
}

package repoctl

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestUpdateReadmeSizeFromBenchmark(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "benchmarks.json"), []byte(`{"binary":{"size_mb":"16.2"}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	readme := filepath.Join(root, "README.md")
	if err := os.WriteFile(readme, []byte("Binary size: ~99.9 MB\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := UpdateReadme(ReadmeOptions{Root: root}); err != nil {
		t.Fatalf("UpdateReadme: %v", err)
	}
	raw, err := os.ReadFile(readme)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "~16.2 MB") {
		t.Fatalf("README not updated:\n%s", raw)
	}
}

func TestUpdateReadmeSizeFromNumericBenchmark(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "benchmarks.json"), []byte(`{"binary":{"size_mb":16.2}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	readme := filepath.Join(root, "README.md")
	if err := os.WriteFile(readme, []byte("Binary size: ~99.9 MB\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := UpdateReadme(ReadmeOptions{Root: root}); err != nil {
		t.Fatalf("UpdateReadme: %v", err)
	}
	raw, err := os.ReadFile(readme)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "~16.2 MB") {
		t.Fatalf("README not updated:\n%s", raw)
	}
}

func TestUpdateReadmeSkipsMissingBenchmarks(t *testing.T) {
	root := t.TempDir()
	readme := filepath.Join(root, "README.md")
	original := "Binary size: ~99.9 MB\n"
	if err := os.WriteFile(readme, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := UpdateReadme(ReadmeOptions{Root: root}); err != nil {
		t.Fatalf("UpdateReadme: %v", err)
	}
	raw, err := os.ReadFile(readme)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != original {
		t.Fatalf("README changed:\n%s", raw)
	}
}

func TestREADMEStartsWithMethodology(t *testing.T) {
	raw := readRepoFile(t, "README.md")
	lead := firstReadmeBodyParagraph(raw)
	if !strings.Contains(lead, "autonomous-porting methodology") {
		t.Fatalf("README lead should start with the autonomous-porting methodology, got %q", lead)
	}
	if strings.Contains(strings.ToLower(lead), "go port") {
		t.Fatalf("README lead should not frame Gormes as a Go port: %q", lead)
	}
}

func TestREADMEMentionsDifferentiator(t *testing.T) {
	raw := readRepoFile(t, "README.md")
	firstWords := firstNWords(markdownWords(raw), 50)
	wants := []string{
		"30 most-used Hermes skills",
		"single 30 MB Go binary",
		"Termux",
		"Windows-without-Python",
		"locked-down corp Linux",
	}
	for _, want := range wants {
		if !strings.Contains(firstWords, want) {
			t.Fatalf("README first 50 words should include %q; got %q", want, firstWords)
		}
	}
}

func TestREADMEPreservesOperatorSections(t *testing.T) {
	raw := readRepoFile(t, "README.md")
	wants := []string{
		"## Quick Install",
		"## First Run",
		"## Status",
		"curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/install.sh | bash",
		"irm https://gormes.ai/install.ps1 | iex",
		"gormes doctor --offline",
		"gormes --offline",
	}
	for _, want := range wants {
		if !strings.Contains(raw, want) {
			t.Fatalf("README must preserve operator section or command %q", want)
		}
	}
}

func TestREADMELinksStrategyDoc(t *testing.T) {
	raw := readRepoFile(t, "README.md")
	want := "docs/content/building-gormes/strategy/success-plan.md"
	if !strings.Contains(raw, want) {
		t.Fatalf("README should link the strategy doc %q", want)
	}
}

func readRepoFile(t *testing.T, rel string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	raw, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func firstReadmeBodyParagraph(raw string) string {
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "<") || strings.HasPrefix(line, "![") || strings.HasPrefix(line, "---") {
			continue
		}
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[!") || strings.HasPrefix(line, "`") {
			continue
		}
		return line
	}
	return ""
}

func markdownWords(raw string) string {
	replacements := []string{"<br>", " ", "</strong>", " ", "<strong>", " "}
	cleaned := strings.NewReplacer(replacements...).Replace(raw)
	cleaned = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(cleaned, " ")
	cleaned = regexp.MustCompile(`!\[[^\]]*\]\([^)]+\)`).ReplaceAllString(cleaned, " ")
	cleaned = regexp.MustCompile(`\[[^\]]+\]\([^)]+\)`).ReplaceAllStringFunc(cleaned, func(match string) string {
		close := strings.Index(match, "]")
		if close == -1 {
			return " "
		}
		return match[1:close]
	})
	cleaned = strings.NewReplacer("`", "", "*", "", "_", "", "#", "", "|", " ").Replace(cleaned)
	return strings.Join(strings.Fields(cleaned), " ")
}

func firstNWords(raw string, n int) string {
	words := strings.Fields(raw)
	if len(words) > n {
		words = words[:n]
	}
	return strings.Join(words, " ")
}

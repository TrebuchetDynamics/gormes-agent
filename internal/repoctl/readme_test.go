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

func TestUpdateReadmeSyncsReleaseAndBenchmarkMetadata(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "benchmarks.json"), []byte(`{
  "binary": {
    "size_mb": "28.0",
    "last_measured": "2026-05-09"
  },
  "code": {
    "test_count": 4603,
    "go_files": 748,
    "go_lines": 185311,
    "dependencies": 139
  }
}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	releasePath := filepath.Join(root, "webpages", "landing", "src", "data", "release.json")
	if err := os.MkdirAll(filepath.Dir(releasePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(releasePath, []byte(`{
  "version": "0.2.0",
  "tag": "v0.2.0",
  "url": "https://github.com/TrebuchetDynamics/gormes-agent/releases/tag/v0.2.0"
}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	readme := filepath.Join(root, "README.md")
	original := strings.Join([]string{
		"Gormes runs the 30 most-used Hermes skills unchanged in a single 30 MB Go binary.",
		"",
		"Latest public release: [v0.1.05](https://github.com/TrebuchetDynamics/gormes-agent/releases/tag/v0.1.05).",
		"",
		"Current `development` head after `v0.1.05` also includes fixture-backed work.",
		"",
		"Release v0.1.05 publishes static Go binaries for Linux, macOS, and Windows on amd64/arm64. The current benchmark mirror reports a Linux build at ~39.1 MB (`benchmarks.json`, 2026-05-05). CI runs `go test ./... -count=1`, `go run ./cmd/progress validate`, and `git diff --check`.",
		"",
		"1,000+ tests across 10+ Go source files, 20 lines of Go, and 3 dependencies.",
	}, "\n")
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
	got := string(raw)
	wants := []string{
		"single 30 MB Go binary",
		"Latest public release: [v0.2.0](https://github.com/TrebuchetDynamics/gormes-agent/releases/tag/v0.2.0).",
		"Current `development` head after `v0.2.0`",
		"Release v0.2.0 publishes static Go binaries for Linux, macOS, Windows, and Termux/Android across the supported release matrix.",
		"~28.0 MB (`benchmarks.json`, 2026-05-09)",
		"4603+ tests",
		"748+ Go source files",
		"185311 lines of Go",
		"139 dependencies",
	}
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Fatalf("README missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "v0.1.05") || strings.Contains(got, "2026-05-05") || strings.Contains(got, "~39.1 MB") {
		t.Fatalf("README retained stale release or benchmark data:\n%s", got)
	}
}

func TestUpdateReadmeSyncsSTTBenchmarkSummary(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "benchmarks.json"), []byte(`{
  "binary": {
    "size_mb": "42.2",
    "last_measured": "2026-05-10"
  },
  "stt": {
    "wasi_whisper": {
      "last_measured": "2026-05-10",
      "models": [
        {
          "name": "ggml-tiny.en",
          "realtime_factor": 0.92,
          "model_load_ms": 1234,
          "inference_ms": 4567
        }
      ]
    }
  }
}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	readme := filepath.Join(root, "README.md")
	original := strings.Join([]string{
		"## Status",
		"",
		"The current Linux build measures ~28.0 MB (`benchmarks.json`). WASI Whisper tiny.en has not been benchmarked yet.",
	}, "\n")
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
	got := string(raw)
	wants := []string{
		"The current Linux build measures ~42.2 MB (`benchmarks.json`).",
		"WASI Whisper tiny.en runs at 0.92x realtime (`benchmarks.json`, 2026-05-10).",
	}
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Fatalf("README missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "has not been benchmarked yet") || strings.Contains(got, "~28.0 MB") {
		t.Fatalf("README retained stale STT benchmark text:\n%s", got)
	}
}

// TestREADMELeadsWithRuntime pins the runtime-first hero contract.
// Earlier drafts led with "autonomous-porting methodology" — the
// 2026-05-09 README rebalance walked that back: README opens with
// the runtime story, methodology lives in a dedicated section near
// the bottom as supporting evidence. The site/landing copy may
// still lead with methodology; this assertion is README-only.
func TestREADMELeadsWithRuntime(t *testing.T) {
	raw := readRepoFile(t, "README.md")
	lead := firstReadmeBodyParagraph(raw)
	leadLower := strings.ToLower(lead)
	for _, banned := range []string{"go port"} {
		if strings.Contains(leadLower, banned) {
			t.Fatalf("README lead should not frame Gormes as a Go port: %q", lead)
		}
	}
	if !strings.Contains(leadLower, "go binary") && !strings.Contains(leadLower, "go-native") {
		t.Fatalf("README lead must anchor on the Go-native runtime story; got %q", lead)
	}
	// Methodology must still appear somewhere — just not in the
	// lead. The dedicated section lives further down.
	if !strings.Contains(raw, "autonomous-porting methodology") {
		t.Fatalf("README must still mention the autonomous-porting methodology somewhere (relocated to its own section, not the lead)")
	}
}

// TestREADMEMentionsDifferentiator asserts the v1 differentiator
// spec lands in the README's hero block (within the first 80 words):
// the 30 most-used Hermes skills running on Termux,
// Windows-without-Python, and locked-down corp Linux. Previously
// this also required "single 30 MB Go binary" in the first 50
// words — the rebalance moved binary-size into the Status section
// to keep the hero focused on portability + skill count.
func TestREADMEMentionsDifferentiator(t *testing.T) {
	raw := readRepoFile(t, "README.md")
	firstWords := firstNWords(markdownWords(raw), 80)
	wants := []string{
		"30 most-used Hermes skills",
		"Termux",
		"Windows-without-Python",
		"locked-down corp Linux",
	}
	for _, want := range wants {
		if !strings.Contains(firstWords, want) {
			t.Fatalf("README first 80 words should include %q; got %q", want, firstWords)
		}
	}
}

// TestREADMEPreservesOperatorSections asserts the runtime-first
// sections operators rely on. Section names tracked here: Quick
// Install (entry point), First Proof (the rebalanced "First Run" —
// renamed to lean into doctor/--offline as the proof), Status (the
// release/CI summary). The two install-script invocations and the
// two offline-proof commands round out the contract.
func TestREADMEPreservesOperatorSections(t *testing.T) {
	raw := readRepoFile(t, "README.md")
	wants := []string{
		"## Quick Install",
		"## First Proof",
		"## Status",
		"curl -fsSL https://gormes.ai/install.sh | sh",
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

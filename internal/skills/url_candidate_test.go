package skills

import (
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestURLSkillCandidate_FromFrontmatter(t *testing.T) {
	raw := testURLSkillDocument("name: sharethis-chat")

	got, err := ParseURLSkillCandidate("https://example.com/tools/not-used/SKILL.md", []byte(raw))
	if err != nil {
		t.Fatalf("ParseURLSkillCandidate returned unexpected error: %v", err)
	}

	if got.Name != "sharethis-chat" {
		t.Fatalf("Name = %q, want %q", got.Name, "sharethis-chat")
	}
	if got.AwaitingName {
		t.Fatal("AwaitingName = true, want false")
	}
	if got.Source != "url" {
		t.Fatalf("Source = %q, want %q", got.Source, "url")
	}
	if got.Trust != "community" {
		t.Fatalf("Trust = %q, want %q", got.Trust, "community")
	}
	assertURLCandidateFiles(t, got.Files, raw)
	if got.Evidence != "" {
		t.Fatalf("Evidence = %q, want empty", got.Evidence)
	}
}

func TestURLSkillCandidate_FromURLSlug(t *testing.T) {
	raw := testURLSkillDocument("")

	got, err := ParseURLSkillCandidate("https://example.com/tools/review-bot/SKILL.md", []byte(raw))
	if err != nil {
		t.Fatalf("ParseURLSkillCandidate returned unexpected error: %v", err)
	}

	if got.Name != "review-bot" {
		t.Fatalf("Name = %q, want %q", got.Name, "review-bot")
	}
	if got.AwaitingName {
		t.Fatal("AwaitingName = true, want false")
	}
	if got.Evidence != "" {
		t.Fatalf("Evidence = %q, want empty", got.Evidence)
	}
	assertURLCandidateFiles(t, got.Files, raw)
}

func TestURLSkillCandidate_MissingNameEvidence(t *testing.T) {
	raw := testURLSkillDocument("")
	rawURL := "https://example.com/SKILL.md"

	got, err := ParseURLSkillCandidate(rawURL, []byte(raw))
	if err != nil {
		t.Fatalf("ParseURLSkillCandidate returned unexpected error: %v", err)
	}

	if got.Name != "" {
		t.Fatalf("Name = %q, want empty", got.Name)
	}
	if !got.AwaitingName {
		t.Fatal("AwaitingName = false, want true")
	}
	if got.Evidence != URLSkillEvidenceMissingName {
		t.Fatalf("Evidence = %q, want %q", got.Evidence, URLSkillEvidenceMissingName)
	}
	if !strings.Contains(got.RetryHint, "gormes skills install "+rawURL+" --name <your-name>") {
		t.Fatalf("RetryHint = %q, want gormes retry guidance", got.RetryHint)
	}
	assertURLCandidateFiles(t, got.Files, raw)
}

func TestURLSkillCandidate_RejectsUnsafeNames(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter string
	}{
		{name: "sentinel skill", frontmatter: "name: skill"},
		{name: "sentinel readme", frontmatter: "name: README"},
		{name: "sentinel index", frontmatter: "name: index"},
		{name: "sentinel unnamed", frontmatter: "name: unnamed-skill"},
		{name: "nested", frontmatter: "name: tools/review-bot"},
		{name: "absolute", frontmatter: "name: /review-bot"},
		{name: "drive letter", frontmatter: `name: "C:"`},
		{name: "drive path", frontmatter: "name: C:/review-bot"},
		{name: "dotdot", frontmatter: "name: .."},
		{name: "traversal", frontmatter: "name: ../review-bot"},
		{name: "nested traversal", frontmatter: "name: safe/../review-bot"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseURLSkillCandidate("https://example.com/tools/fallback-safe/SKILL.md", []byte(testURLSkillDocument(tc.frontmatter)))
			if err == nil {
				t.Fatal("ParseURLSkillCandidate returned nil error, want unsafe name rejection")
			}
			if got.Evidence != URLSkillEvidenceInvalidName {
				t.Fatalf("Evidence = %q, want %q", got.Evidence, URLSkillEvidenceInvalidName)
			}
			if got.Name != "" {
				t.Fatalf("Name = %q, want empty when rejecting unsafe name", got.Name)
			}
			if len(got.Files) != 0 {
				t.Fatalf("Files = %v, want empty before returning an unsafe path", got.Files)
			}
		})
	}
}

func TestURLSkillCandidate_RejectsInvalidURL(t *testing.T) {
	tests := []string{
		"",
		"not-a-url",
		"http://example.com/tools/review-bot/SKILL.md",
		"https://example.com/tools/review-bot/README.md",
		"https://example.com/tools/review-bot/",
		"https://example.com/.well-known/skills/review-bot/SKILL.md",
		"https://example.com/tools/../review-bot/SKILL.md",
	}

	for _, rawURL := range tests {
		t.Run(rawURL, func(t *testing.T) {
			got, err := ParseURLSkillCandidate(rawURL, []byte(testURLSkillDocument("name: review-bot")))
			if err == nil {
				t.Fatal("ParseURLSkillCandidate returned nil error, want invalid URL rejection")
			}
			if got.Evidence != URLSkillEvidenceInvalidURL {
				t.Fatalf("Evidence = %q, want %q", got.Evidence, URLSkillEvidenceInvalidURL)
			}
			if got.Name != "" {
				t.Fatalf("Name = %q, want empty", got.Name)
			}
			if len(got.Files) != 0 {
				t.Fatalf("Files = %v, want empty", got.Files)
			}
		})
	}
}

func TestURLSkillCandidate_InvalidFrontmatterEvidence(t *testing.T) {
	tests := map[string]string{
		"missing delimiters": "name: review-bot\n\n# Review Bot\n",
		"malformed yaml":     "---\nname: [unterminated\n---\n\n# Review Bot\n",
		"missing body":       "---\nname: review-bot\ndescription: Reviews code\n---\n",
	}

	for name, raw := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ParseURLSkillCandidate("https://example.com/tools/review-bot/SKILL.md", []byte(raw))
			if err == nil {
				t.Fatal("ParseURLSkillCandidate returned nil error, want invalid frontmatter rejection")
			}
			if got.Evidence != URLSkillEvidenceInvalidFrontmatter {
				t.Fatalf("Evidence = %q, want %q", got.Evidence, URLSkillEvidenceInvalidFrontmatter)
			}
			if len(got.Files) != 0 {
				t.Fatalf("Files = %v, want empty", got.Files)
			}
		})
	}
}

func TestURLSkillCandidate_NoNetworkOrFilesystemImports(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "url_candidate.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("ParseFile returned error: %v", err)
	}

	forbidden := map[string]bool{
		`"internal/cli"`:  true,
		`"net/http"`:      true,
		`"os"`:            true,
		`"path/filepath"`: true,
	}
	for _, imported := range file.Imports {
		if forbidden[imported.Path.Value] {
			t.Fatalf("url_candidate.go imports %s; helper must stay pure", imported.Path.Value)
		}
	}

	raw, err := os.ReadFile("url_candidate.go")
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	source := string(raw)
	for _, forbiddenCall := range []string{"WriteFile", "MkdirAll", "Create(", "RemoveAll"} {
		if strings.Contains(source, forbiddenCall) {
			t.Fatalf("url_candidate.go contains %q; helper must not write filesystem state", forbiddenCall)
		}
	}
}

func testURLSkillDocument(frontmatterName string) string {
	lines := []string{
		"---",
	}
	if strings.TrimSpace(frontmatterName) != "" {
		lines = append(lines, frontmatterName)
	}
	lines = append(lines,
		"description: Reviews collaboration artifacts",
		"---",
		"",
		"# Review Bot",
		"",
		"Review the supplied artifacts and report actionable findings.",
		"",
	)
	return strings.Join(lines, "\n")
}

func assertURLCandidateFiles(t *testing.T, got map[string][]byte, wantSkillMD string) {
	t.Helper()

	want := map[string][]byte{"SKILL.md": []byte(wantSkillMD)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Files = %#v, want exactly SKILL.md", got)
	}
}

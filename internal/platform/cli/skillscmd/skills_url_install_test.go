package skillscmd

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/spf13/cobra"
)

// fakeURLFetcher records every fetch and returns a fixed payload.
type fakeURLFetcher struct {
	calls   []string
	payload []byte
}

func (f *fakeURLFetcher) Fetch(_ context.Context, url string) ([]byte, error) {
	f.calls = append(f.calls, url)
	return append([]byte(nil), f.payload...), nil
}

// fakeURLScanner returns a fixed verdict.
type fakeURLScanner struct {
	ok       bool
	evidence string
	calls    int
}

func (s *fakeURLScanner) Scan(_ context.Context, _ []byte) (bool, string, error) {
	s.calls++
	return s.ok, s.evidence, nil
}

// fakeURLStore captures every WriteSkill call (path, body) for assertions.
type fakeURLStore struct {
	root  string
	files map[string][]byte
}

func newFakeURLStore(root string) *fakeURLStore {
	return &fakeURLStore{root: root, files: map[string][]byte{}}
}

func (m *fakeURLStore) ActiveDir() string { return filepath.Join(m.root, "active") }
func (m *fakeURLStore) WriteSkill(_ context.Context, dir string, file string, body []byte) (string, error) {
	full := filepath.Join(dir, file)
	m.files[full] = append([]byte(nil), body...)
	return full, nil
}

func skillURLDoc(name string) []byte {
	lines := []string{"---"}
	if strings.TrimSpace(name) != "" {
		lines = append(lines, "name: "+name)
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
	return []byte(strings.Join(lines, "\n"))
}

func newURLInstallTestCmd(t *testing.T, fetcher skills.URLFetcher, scanner skills.QuarantineScanner, store skills.SkillStore) *cobra.Command {
	t.Helper()
	deps := SkillsCommandDeps{
		ListInstalledSkills: func(skills.ListOptions, map[string]struct{}) []skills.SkillRow { return nil },
		DisabledSkills:      func(string) map[string]struct{} { return nil },
		URLInstall: SkillsURLInstallDeps{
			Fetcher: fetcher,
			Scanner: scanner,
			Store:   store,
		},
	}
	return NewSkillsCommand(deps)
}

func executeURLInstall(cmd *cobra.Command, args ...string) (string, error) {
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), err
}

func TestSkillsURLInstall_NonInteractiveRequiresName(t *testing.T) {
	rawURL := "https://example.com/SKILL.md"
	fetcher := &fakeURLFetcher{payload: skillURLDoc("")}
	scanner := &fakeURLScanner{ok: true, evidence: "scan_clean"}
	store := newFakeURLStore(t.TempDir())

	cmd := newURLInstallTestCmd(t, fetcher, scanner, store)

	stdout, err := executeURLInstall(cmd, "install", rawURL)
	if err == nil {
		t.Fatal("Execute() returned nil error, want missing-name failure")
	}

	wantHint := "gormes skills install " + rawURL + " --name <your-name>"
	if !strings.Contains(stdout, wantHint) {
		t.Fatalf("stdout missing retry guidance %q:\n%s", wantHint, stdout)
	}
	if len(store.files) != 0 {
		t.Fatalf("store wrote %d files, want 0", len(store.files))
	}
}

// TestSkillsURLInstall_JSONEmitsStructuredInstallEvent proves
// `gormes skills install <url> --json` returns a parseable
// `{build, action, name, installed_path}` document so fleet automation
// rolling out skills across machines can confirm where each install
// landed without scraping the "installed <path>" prose. Build
// provenance leads — same convention as `skills list --json` (slice
// 105) and the rest of the JSON arc.
func TestSkillsURLInstall_JSONEmitsStructuredInstallEvent(t *testing.T) {
	rawURL := "https://example.com/SKILL.md"
	fetcher := &fakeURLFetcher{payload: skillURLDoc("")}
	scanner := &fakeURLScanner{ok: true, evidence: "scan_clean"}
	store := newFakeURLStore(t.TempDir())

	deps := SkillsCommandDeps{
		ListInstalledSkills: func(skills.ListOptions, map[string]struct{}) []skills.SkillRow { return nil },
		DisabledSkills:      func(string) map[string]struct{} { return nil },
		URLInstall: SkillsURLInstallDeps{
			Fetcher: fetcher,
			Scanner: scanner,
			Store:   store,
		},
		BuildProvenance: func() any {
			return map[string]string{"version": "test-install-attr"}
		},
	}
	cmd := NewSkillsCommand(deps)

	stdout, err := executeURLInstall(cmd, "install", rawURL, "--name", "my-url-skill", "--json")
	if err != nil {
		t.Fatalf("Execute(): %v\nstdout: %s", err, stdout)
	}
	var got struct {
		Build         map[string]string `json:"build"`
		Action        string            `json:"action"`
		InstalledPath string            `json:"installed_path"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("invalid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build["version"] != "test-install-attr" {
		t.Errorf("build.version = %q, want test-install-attr", got.Build["version"])
	}
	if got.Action != "installed" {
		t.Errorf("action = %q, want installed", got.Action)
	}
	if got.InstalledPath == "" {
		t.Errorf("installed_path empty, want populated")
	}
}

func TestSkillsURLInstall_NameOverride(t *testing.T) {
	rawURL := "https://example.com/SKILL.md"
	fetcher := &fakeURLFetcher{payload: skillURLDoc("")}
	scanner := &fakeURLScanner{ok: true, evidence: "scan_clean"}
	store := newFakeURLStore(t.TempDir())

	cmd := newURLInstallTestCmd(t, fetcher, scanner, store)

	stdout, err := executeURLInstall(cmd, "install", rawURL, "--name", "my-url-skill")
	if err != nil {
		t.Fatalf("Execute() error = %v\nstdout: %s", err, stdout)
	}
	if len(fetcher.calls) != 1 {
		t.Fatalf("fetcher.calls = %d, want 1", len(fetcher.calls))
	}
	if scanner.calls != 1 {
		t.Fatalf("scanner.calls = %d, want 1 (must scan before write)", scanner.calls)
	}
	if len(store.files) != 1 {
		t.Fatalf("store.files = %d, want exactly 1", len(store.files))
	}
	for path := range store.files {
		if filepath.Base(path) != "SKILL.md" {
			t.Fatalf("written path %q does not end in SKILL.md", path)
		}
		if !strings.Contains(path, "my-url-skill") {
			t.Fatalf("written path %q does not include override name", path)
		}
	}
}

func TestSkillsURLInstall_InvalidNameOverride(t *testing.T) {
	cases := []struct {
		label    string
		override string
	}{
		{"sentinel SKILL", "SKILL"},
		{"nested path", "a/b"},
		{"traversal dotdot", ".."},
		{"nested traversal", "a/../b"},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			fetcher := &fakeURLFetcher{payload: skillURLDoc("")}
			scanner := &fakeURLScanner{ok: true}
			store := newFakeURLStore(t.TempDir())

			cmd := newURLInstallTestCmd(t, fetcher, scanner, store)

			_, err := executeURLInstall(cmd, "install", "https://example.com/SKILL.md", "--name", tc.override)
			if err == nil {
				t.Fatalf("Execute() returned nil error for %q, want invalid name rejection", tc.override)
			}
			if len(fetcher.calls) != 0 {
				t.Fatalf("fetcher.calls = %d, want 0 (rejection must precede fetch)", len(fetcher.calls))
			}
			if len(store.files) != 0 {
				t.Fatalf("store.files = %d, want 0", len(store.files))
			}
		})
	}
}

func TestSkillsURLInstall_CategoryGuard(t *testing.T) {
	rawURL := "https://example.com/sharethis-chat/SKILL.md"

	t.Run("accepts productivity", func(t *testing.T) {
		fetcher := &fakeURLFetcher{payload: skillURLDoc("sharethis-chat")}
		scanner := &fakeURLScanner{ok: true}
		store := newFakeURLStore(t.TempDir())

		cmd := newURLInstallTestCmd(t, fetcher, scanner, store)

		_, err := executeURLInstall(cmd, "install", rawURL, "--category", "productivity")
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if len(store.files) != 1 {
			t.Fatalf("store.files = %d, want 1", len(store.files))
		}
		for path := range store.files {
			if !strings.Contains(path, "productivity") {
				t.Fatalf("written path %q does not include productivity", path)
			}
		}
	})

	rejections := []string{"/etc", "..", "foo/../bar"}
	for _, cat := range rejections {
		t.Run("rejects "+cat, func(t *testing.T) {
			fetcher := &fakeURLFetcher{payload: skillURLDoc("sharethis-chat")}
			scanner := &fakeURLScanner{ok: true}
			store := newFakeURLStore(t.TempDir())

			cmd := newURLInstallTestCmd(t, fetcher, scanner, store)

			_, err := executeURLInstall(cmd, "install", rawURL, "--category", cat)
			if err == nil {
				t.Fatalf("Execute() returned nil error for category %q", cat)
			}
			if len(fetcher.calls) != 0 {
				t.Fatalf("fetcher.calls = %d, want 0 (category rejection precedes fetch)", len(fetcher.calls))
			}
			if len(store.files) != 0 {
				t.Fatalf("store.files = %d, want 0", len(store.files))
			}
		})
	}
}

func TestSkillsURLInstall_NoLiveHTTP(t *testing.T) {
	rawURL := "https://example.com/tools/review-bot/SKILL.md"
	fetcher := &fakeURLFetcher{payload: skillURLDoc("review-bot")}
	scanner := &fakeURLScanner{ok: true}
	store := newFakeURLStore(t.TempDir())

	cmd := newURLInstallTestCmd(t, fetcher, scanner, store)

	if _, err := executeURLInstall(cmd, "install", rawURL); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(fetcher.calls) != 1 {
		t.Fatalf("fetcher.calls = %d, want 1", len(fetcher.calls))
	}

	// Rejected install: zero fetch calls.
	fetcher2 := &fakeURLFetcher{payload: skillURLDoc("")}
	scanner2 := &fakeURLScanner{ok: true}
	store2 := newFakeURLStore(t.TempDir())
	cmd2 := newURLInstallTestCmd(t, fetcher2, scanner2, store2)
	if _, err := executeURLInstall(cmd2, "install", "https://example.com/SKILL.md", "--name", "SKILL"); err == nil {
		t.Fatal("Execute() returned nil error for invalid name")
	}
	if len(fetcher2.calls) != 0 {
		t.Fatalf("fetcher2.calls = %d, want 0", len(fetcher2.calls))
	}
}

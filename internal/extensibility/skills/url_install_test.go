package skills

import (
	"context"
	"errors"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeFetcher records every Fetch call and returns the configured payload.
type fakeFetcher struct {
	calls    []string
	payload  []byte
	errOnURL map[string]error
}

func (f *fakeFetcher) Fetch(_ context.Context, url string) ([]byte, error) {
	f.calls = append(f.calls, url)
	if err, ok := f.errOnURL[url]; ok {
		return nil, err
	}
	return append([]byte(nil), f.payload...), nil
}

// fakeScanner returns a fixed verdict for any payload.
type fakeScanner struct {
	ok       bool
	evidence string
	err      error
	calls    int
}

func (s *fakeScanner) Scan(_ context.Context, _ []byte) (bool, string, error) {
	s.calls++
	return s.ok, s.evidence, s.err
}

// memorySkillStore is an in-memory SkillStore that records every WriteSkill
// call so tests can assert exactly one path was written, and inspect that
// path/payload to prove no live filesystem mutation outside t.TempDir().
type memorySkillStore struct {
	root  string
	files map[string][]byte
}

func newMemorySkillStore(root string) *memorySkillStore {
	return &memorySkillStore{root: root, files: map[string][]byte{}}
}

func (m *memorySkillStore) ActiveDir() string {
	return filepath.Join(m.root, "active")
}

func (m *memorySkillStore) WriteSkill(_ context.Context, dir string, file string, body []byte) (string, error) {
	full := filepath.Join(dir, file)
	m.files[full] = append([]byte(nil), body...)
	return full, nil
}

func urlInstallTestDoc(name string) []byte {
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

func TestURLSkillInstall_NonInteractiveRequiresName(t *testing.T) {
	rawURL := "https://example.com/SKILL.md" // produces awaiting_name
	fetcher := &fakeFetcher{payload: urlInstallTestDoc("")}
	scanner := &fakeScanner{ok: true, evidence: "scan_clean"}
	store := newMemorySkillStore(t.TempDir())

	policy := URLInstallPolicy{
		Fetcher: fetcher,
		Scanner: scanner,
		Store:   store,
	}

	got := PerformURLInstall(context.Background(), policy, URLInstallRequest{
		URL:         rawURL,
		Interactive: false,
	})

	if got.Code != string(URLSkillEvidenceMissingName) {
		t.Fatalf("Code = %q, want %q", got.Code, URLSkillEvidenceMissingName)
	}
	wantHint := "gormes skills install " + rawURL + " --name <your-name>"
	if !strings.Contains(got.Reason, wantHint) {
		t.Fatalf("Reason = %q, want substring %q", got.Reason, wantHint)
	}
	if got.InstalledPath != "" {
		t.Fatalf("InstalledPath = %q, want empty when missing name", got.InstalledPath)
	}
	if len(store.files) != 0 {
		t.Fatalf("store wrote %d files, want 0 on missing-name", len(store.files))
	}
	// active store must be unchanged: scan substring proves no SKILL.md path exists.
	if walked := walkContainsSkill(t, store.ActiveDir()); walked {
		t.Fatalf("active dir %q contains SKILL.md; want nothing written", store.ActiveDir())
	}
}

func TestURLSkillInstall_NameOverride(t *testing.T) {
	rawURL := "https://example.com/SKILL.md"
	fetcher := &fakeFetcher{payload: urlInstallTestDoc("")}
	scanner := &fakeScanner{ok: true, evidence: "scan_clean"}
	store := newMemorySkillStore(t.TempDir())

	policy := URLInstallPolicy{Fetcher: fetcher, Scanner: scanner, Store: store}

	got := PerformURLInstall(context.Background(), policy, URLInstallRequest{
		URL:          rawURL,
		NameOverride: "my-url-skill",
		Interactive:  false,
	})

	if got.Code != "url_skill_installed" {
		t.Fatalf("Code = %q, want url_skill_installed; reason=%q", got.Code, got.Reason)
	}
	if got.InstalledPath == "" {
		t.Fatalf("InstalledPath empty, want set; got=%+v", got)
	}
	if len(fetcher.calls) != 1 {
		t.Fatalf("fetcher.calls = %d, want exactly 1", len(fetcher.calls))
	}
	if scanner.calls != 1 {
		t.Fatalf("scanner.calls = %d, want exactly 1 (must stage through scan)", scanner.calls)
	}
	if len(store.files) != 1 {
		t.Fatalf("store.files = %d, want exactly 1", len(store.files))
	}
	for path := range store.files {
		if filepath.Base(path) != "SKILL.md" {
			t.Fatalf("written file %q does not end in SKILL.md", path)
		}
		if !strings.Contains(path, "my-url-skill") {
			t.Fatalf("written path %q does not include override name my-url-skill", path)
		}
	}
}

func TestURLSkillInstall_InvalidNameOverride(t *testing.T) {
	rawURL := "https://example.com/SKILL.md"

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
			fetcher := &fakeFetcher{payload: urlInstallTestDoc("")}
			scanner := &fakeScanner{ok: true, evidence: "scan_clean"}
			store := newMemorySkillStore(t.TempDir())

			policy := URLInstallPolicy{Fetcher: fetcher, Scanner: scanner, Store: store}

			got := PerformURLInstall(context.Background(), policy, URLInstallRequest{
				URL:          rawURL,
				NameOverride: tc.override,
				Interactive:  false,
			})

			if got.Code != string(URLSkillEvidenceInvalidName) {
				t.Fatalf("Code = %q, want %q (override=%q)", got.Code, URLSkillEvidenceInvalidName, tc.override)
			}
			if got.InstalledPath != "" {
				t.Fatalf("InstalledPath = %q, want empty for invalid name", got.InstalledPath)
			}
			if len(fetcher.calls) != 0 {
				t.Fatalf("fetcher.calls = %d, want 0 (validation must precede fetch)", len(fetcher.calls))
			}
			if scanner.calls != 0 {
				t.Fatalf("scanner.calls = %d, want 0", scanner.calls)
			}
			if len(store.files) != 0 {
				t.Fatalf("store wrote %d files, want 0 on invalid name", len(store.files))
			}
		})
	}
}

func TestURLSkillInstall_CategoryGuard(t *testing.T) {
	rawURL := "https://example.com/sharethis-chat/SKILL.md"

	t.Run("accepts productivity", func(t *testing.T) {
		fetcher := &fakeFetcher{payload: urlInstallTestDoc("sharethis-chat")}
		scanner := &fakeScanner{ok: true, evidence: "scan_clean"}
		store := newMemorySkillStore(t.TempDir())

		policy := URLInstallPolicy{Fetcher: fetcher, Scanner: scanner, Store: store}

		got := PerformURLInstall(context.Background(), policy, URLInstallRequest{
			URL:              rawURL,
			CategoryOverride: "productivity",
			Interactive:      false,
		})

		if got.Code != "url_skill_installed" {
			t.Fatalf("Code = %q, want url_skill_installed; reason=%q", got.Code, got.Reason)
		}
		if !strings.Contains(got.InstalledPath, "productivity") {
			t.Fatalf("InstalledPath = %q, want it under productivity/", got.InstalledPath)
		}
	})

	rejections := []struct {
		label    string
		category string
	}{
		{"absolute path", "/etc"},
		{"traversal dotdot", ".."},
		{"nested traversal", "foo/../bar"},
	}
	for _, tc := range rejections {
		t.Run("rejects "+tc.label, func(t *testing.T) {
			fetcher := &fakeFetcher{payload: urlInstallTestDoc("sharethis-chat")}
			scanner := &fakeScanner{ok: true, evidence: "scan_clean"}
			store := newMemorySkillStore(t.TempDir())

			policy := URLInstallPolicy{Fetcher: fetcher, Scanner: scanner, Store: store}

			got := PerformURLInstall(context.Background(), policy, URLInstallRequest{
				URL:              rawURL,
				CategoryOverride: tc.category,
				Interactive:      false,
			})

			if got.Code != "url_skill_invalid_category" {
				t.Fatalf("Code = %q, want url_skill_invalid_category for %q", got.Code, tc.category)
			}
			if got.InstalledPath != "" {
				t.Fatalf("InstalledPath = %q, want empty on category rejection", got.InstalledPath)
			}
			if len(fetcher.calls) != 0 {
				t.Fatalf("fetcher.calls = %d, want 0 (category validation must precede fetch)", len(fetcher.calls))
			}
			if len(store.files) != 0 {
				t.Fatalf("store wrote %d files, want 0 on invalid category", len(store.files))
			}
		})
	}
}

func TestURLSkillInstall_NoLiveHTTP(t *testing.T) {
	// Forbid net/http imports in the policy file to prove live HTTP is impossible.
	file, err := parser.ParseFile(token.NewFileSet(), "url_install.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("ParseFile returned error: %v", err)
	}
	for _, imported := range file.Imports {
		if imported.Path.Value == `"net/http"` {
			t.Fatalf("url_install.go imports net/http; live HTTP must be injected through URLFetcher")
		}
	}

	// Fetcher records exactly one call on the happy path.
	fetcher := &fakeFetcher{payload: urlInstallTestDoc("review-bot")}
	scanner := &fakeScanner{ok: true, evidence: "scan_clean"}
	store := newMemorySkillStore(t.TempDir())

	got := PerformURLInstall(context.Background(), URLInstallPolicy{
		Fetcher: fetcher, Scanner: scanner, Store: store,
	}, URLInstallRequest{
		URL:         "https://example.com/tools/review-bot/SKILL.md",
		Interactive: false,
	})
	if got.Code != "url_skill_installed" {
		t.Fatalf("Code = %q, want url_skill_installed", got.Code)
	}
	if len(fetcher.calls) != 1 {
		t.Fatalf("fetcher.calls = %d, want exactly 1 (no live HTTP, all routed through fetcher)", len(fetcher.calls))
	}

	// Rejected-name install records zero calls on the fetcher (validation precedes fetch).
	fetcher2 := &fakeFetcher{payload: urlInstallTestDoc("")}
	scanner2 := &fakeScanner{ok: true}
	store2 := newMemorySkillStore(t.TempDir())
	got2 := PerformURLInstall(context.Background(), URLInstallPolicy{
		Fetcher: fetcher2, Scanner: scanner2, Store: store2,
	}, URLInstallRequest{
		URL:          "https://example.com/SKILL.md",
		NameOverride: "SKILL",
		Interactive:  false,
	})
	if got2.Code != string(URLSkillEvidenceInvalidName) {
		t.Fatalf("Code = %q, want %q", got2.Code, URLSkillEvidenceInvalidName)
	}
	if len(fetcher2.calls) != 0 {
		t.Fatalf("fetcher2.calls = %d, want 0 on rejected name", len(fetcher2.calls))
	}
}

// walkContainsSkill checks if any SKILL.md exists under root.
func walkContainsSkill(t *testing.T, root string) bool {
	t.Helper()
	found := false
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return filepath.SkipAll
			}
			return err
		}
		if !d.IsDir() && filepath.Base(p) == "SKILL.md" {
			found = true
		}
		return nil
	})
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("walk: %v", err)
	}
	return found
}

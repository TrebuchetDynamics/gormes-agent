package doctor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckSkillsHubInitializedCountsHubStateAndToken(t *testing.T) {
	home := t.TempDir()
	hub := filepath.Join(home, "skills", ".hub")
	if err := os.MkdirAll(filepath.Join(hub, "quarantine", "needs-review"), 0o755); err != nil {
		t.Fatalf("mkdir hub quarantine: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hub, "quarantine", "README"), []byte("not a dir"), 0o644); err != nil {
		t.Fatalf("write quarantine file: %v", err)
	}
	lock := `{"installed":{"skill-a":{"version":"1"},"skill-b":{"version":"2"}}}`
	if err := os.WriteFile(filepath.Join(hub, "lock.json"), []byte(lock), 0o644); err != nil {
		t.Fatalf("write lock.json: %v", err)
	}

	called := false
	got := CheckSkillsHub(context.Background(), SkillsHubOptions{
		Home: home,
		Env:  map[string]string{"GITHUB_TOKEN": "ghp_secret_token"},
		RunGHAuthStatus: func(context.Context) GitHubAuthStatusResult {
			called = true
			return GitHubAuthStatusResult{ExitCode: 1}
		},
	})

	if called {
		t.Fatalf("gh runner was called despite GITHUB_TOKEN")
	}
	if got.Name != "Skills Hub" {
		t.Fatalf("Name = %q, want Skills Hub", got.Name)
	}
	if got.Status != StatusWarn {
		t.Fatalf("Status = %v, want WARN because quarantine has pending skill: %+v", got.Status, got)
	}
	items := skillsHubItemsByName(got)
	if it := items[".hub"]; it.Status != StatusPass || !strings.Contains(it.Note, "exists") {
		t.Fatalf(".hub item = %+v, want PASS exists", it)
	}
	if it := items["lock.json"]; it.Status != StatusPass || !strings.Contains(it.Note, "2 hub-installed skill(s)") {
		t.Fatalf("lock item = %+v, want PASS count", it)
	}
	if it := items["quarantine"]; it.Status != StatusWarn || !strings.Contains(it.Note, "1 skill(s) in quarantine") {
		t.Fatalf("quarantine item = %+v, want WARN count", it)
	}
	if it := items["github"]; it.Status != StatusPass || !strings.Contains(it.Note, "GitHub token configured") {
		t.Fatalf("github item = %+v, want token PASS", it)
	}
	out := got.Format()
	for _, forbidden := range []string{"ghp_secret_token", "~/.hermes", "hermes skills", "memories/"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("Skills Hub output leaked forbidden text %q:\n%s", forbidden, out)
		}
	}
}

func TestCheckSkillsHubMissingHubOfflineSkipsGHProbe(t *testing.T) {
	called := false
	got := CheckSkillsHub(context.Background(), SkillsHubOptions{
		Home:    filepath.Join(t.TempDir(), "gormes"),
		Offline: true,
		Env:     map[string]string{},
		RunGHAuthStatus: func(context.Context) GitHubAuthStatusResult {
			called = true
			return GitHubAuthStatusResult{ExitCode: 0}
		},
	})

	if called {
		t.Fatalf("offline Skills Hub check must not run gh auth status")
	}
	if got.Status != StatusWarn {
		t.Fatalf("Status = %v, want WARN because .hub is missing: %+v", got.Status, got)
	}
	items := skillsHubItemsByName(got)
	if it := items[".hub"]; it.Status != StatusWarn || !strings.Contains(it.Note, "gormes skills list") {
		t.Fatalf(".hub missing item = %+v, want WARN with gormes skills list guidance", it)
	}
	if it := items["github"]; it.Status != StatusSkip || !strings.Contains(it.Note, "skipped (--offline") {
		t.Fatalf("github offline item = %+v, want SKIP", it)
	}
	out := got.Format()
	for _, forbidden := range []string{"~/.hermes", "hermes skills"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("Skills Hub output leaked forbidden text %q:\n%s", forbidden, out)
		}
	}
}

func TestCheckSkillsHubCorruptLockWarnsAndGHCLICanPass(t *testing.T) {
	home := t.TempDir()
	hub := filepath.Join(home, "skills", ".hub")
	if err := os.MkdirAll(hub, 0o755); err != nil {
		t.Fatalf("mkdir hub: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hub, "lock.json"), []byte(`{not-json`), 0o644); err != nil {
		t.Fatalf("write corrupt lock.json: %v", err)
	}

	got := CheckSkillsHub(context.Background(), SkillsHubOptions{
		Home: home,
		Env:  map[string]string{},
		RunGHAuthStatus: func(context.Context) GitHubAuthStatusResult {
			return GitHubAuthStatusResult{ExitCode: 0}
		},
	})

	if got.Status != StatusWarn {
		t.Fatalf("Status = %v, want WARN because lock is corrupt: %+v", got.Status, got)
	}
	items := skillsHubItemsByName(got)
	if it := items["lock.json"]; it.Status != StatusWarn || !strings.Contains(it.Note, "corrupted or unreadable") {
		t.Fatalf("lock item = %+v, want WARN corrupt", it)
	}
	if it := items["quarantine"]; it.Status != StatusPass || !strings.Contains(it.Note, "0 skill(s) in quarantine") {
		t.Fatalf("quarantine item = %+v, want PASS zero count", it)
	}
	if it := items["github"]; it.Status != StatusPass || !strings.Contains(it.Note, "gh CLI") {
		t.Fatalf("github item = %+v, want gh CLI PASS", it)
	}
}

func skillsHubItemsByName(r CheckResult) map[string]ItemInfo {
	out := map[string]ItemInfo{}
	for _, it := range r.Items {
		out[it.Name] = it
	}
	return out
}

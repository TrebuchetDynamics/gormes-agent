package stalecode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/stalecodetest"
)

const (
	staleCodeSHA1 = "1111111111111111111111111111111111111111"
	staleCodeSHA2 = "2222222222222222222222222222222222222222"
)

func TestStaleCodeBootSHAEqualCurrentHEADReportsFreshAfterSourceMtimeChange(t *testing.T) {
	root := t.TempDir()
	stalecodetest.WriteNormalGitHEAD(t, root, "main", staleCodeSHA1)
	sourcePath := filepath.Join(root, "internal", "gateway", "status.go")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatalf("create source dir: %v", err)
	}
	if err := os.WriteFile(sourcePath, []byte("first\n"), 0o644); err != nil {
		t.Fatalf("write source fixture: %v", err)
	}

	checker := NewStaleCodeChecker(root)
	got := checker.Check(staleCodeSHA1)
	if got.Status != RuntimeStaleCodeFresh || got.Stale || got.RestartSuggested {
		t.Fatalf("initial stale code = %+v, want fresh", got)
	}

	if err := os.WriteFile(sourcePath, []byte("edited but same HEAD\n"), 0o644); err != nil {
		t.Fatalf("rewrite source fixture: %v", err)
	}
	got = checker.Check(staleCodeSHA1)
	if got.Status != RuntimeStaleCodeFresh || got.Stale || got.RestartSuggested {
		t.Fatalf("after source edit stale code = %+v, want fresh because HEAD did not change", got)
	}
}

func TestStaleCodeChangedHEADReportsRestartEvidence(t *testing.T) {
	root := t.TempDir()
	stalecodetest.WriteNormalGitHEAD(t, root, "development", staleCodeSHA2)

	got := NewStaleCodeChecker(root).Check(staleCodeSHA1)
	if got.Status != RuntimeStaleCodeStale || !got.Stale || !got.RestartSuggested {
		t.Fatalf("stale code = %+v, want stale restart suggestion", got)
	}
	if got.BootGitSHA != staleCodeSHA1 || got.CurrentGitSHA != staleCodeSHA2 {
		t.Fatalf("stale code SHAs = boot %q current %q", got.BootGitSHA, got.CurrentGitSHA)
	}
	assertStaleCodeEvidence(t, got, "stale_code_head_changed")
	assertStaleCodeEvidence(t, got, "stale_code_restart_gateway")
	if strings.Contains(got.Message, root) {
		t.Fatalf("stale code message leaked source root %q in %q", root, got.Message)
	}
}

func TestStaleCodeGitFixturesResolveHEADWithoutSubprocesses(t *testing.T) {
	t.Run("worktree gitdir with commondir", func(t *testing.T) {
		root := t.TempDir()
		worktree := filepath.Join(root, "worktree")
		commonGit := filepath.Join(root, "common.git")
		worktreeGit := filepath.Join(root, "common.git", "worktrees", "wt")
		stalecodetest.WriteFile(t, filepath.Join(worktree, ".git"), "gitdir: "+worktreeGit+"\n")
		stalecodetest.WriteFile(t, filepath.Join(worktreeGit, "HEAD"), "ref: refs/heads/development\n")
		stalecodetest.WriteFile(t, filepath.Join(worktreeGit, "commondir"), "../..\n")
		stalecodetest.WriteFile(t, filepath.Join(commonGit, "refs", "heads", "development"), staleCodeSHA1+"\n")

		got := NewStaleCodeChecker(worktree).Check(staleCodeSHA1)
		if got.Status != RuntimeStaleCodeFresh || got.CurrentGitSHA != staleCodeSHA1 {
			t.Fatalf("worktree stale code = %+v, want fresh resolved through commondir", got)
		}
	})

	t.Run("packed refs", func(t *testing.T) {
		root := t.TempDir()
		stalecodetest.WriteFile(t, filepath.Join(root, ".git", "HEAD"), "ref: refs/heads/development\n")
		stalecodetest.WriteFile(t, filepath.Join(root, ".git", "packed-refs"), "# pack-refs with: peeled fully-peeled sorted\n"+staleCodeSHA1+" refs/heads/development\n")

		got := NewStaleCodeChecker(root).Check(staleCodeSHA1)
		if got.Status != RuntimeStaleCodeFresh || got.CurrentGitSHA != staleCodeSHA1 {
			t.Fatalf("packed ref stale code = %+v, want fresh resolved through packed-refs", got)
		}
	})

	t.Run("detached HEAD", func(t *testing.T) {
		root := t.TempDir()
		stalecodetest.WriteFile(t, filepath.Join(root, ".git", "HEAD"), staleCodeSHA1+"\n")

		got := NewStaleCodeChecker(root).Check(staleCodeSHA1)
		if got.Status != RuntimeStaleCodeFresh || got.CurrentGitSHA != staleCodeSHA1 {
			t.Fatalf("detached stale code = %+v, want fresh detached HEAD", got)
		}
	})
}

func TestStaleCodeNonGitReportsUnavailable(t *testing.T) {
	got := NewStaleCodeChecker(t.TempDir()).Check(staleCodeSHA1)
	if got.Status != RuntimeStaleCodeGitUnavailable || got.Stale || got.RestartSuggested {
		t.Fatalf("non-git stale code = %+v, want unavailable without restart", got)
	}
	assertStaleCodeEvidence(t, got, "stale_code_git_unavailable")
}

func TestStaleCodeCacheRefreshBoundsHEADChanges(t *testing.T) {
	root := t.TempDir()
	stalecodetest.WriteNormalGitHEAD(t, root, "development", staleCodeSHA1)
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	checker := NewStaleCodeChecker(root)
	checker.CacheFreshness = 5 * time.Minute
	checker.now = func() time.Time { return now }

	if got := checker.Check(staleCodeSHA1); got.Status != RuntimeStaleCodeFresh {
		t.Fatalf("initial stale code = %+v, want fresh", got)
	}
	stalecodetest.WriteNormalGitHEAD(t, root, "development", staleCodeSHA2)
	now = now.Add(time.Minute)
	if got := checker.Check(staleCodeSHA1); got.Status != RuntimeStaleCodeFresh || got.CurrentGitSHA != staleCodeSHA1 {
		t.Fatalf("cached stale code = %+v, want cached fresh within freshness window", got)
	}
	now = now.Add(5*time.Minute + time.Second)
	if got := checker.Check(staleCodeSHA1); got.Status != RuntimeStaleCodeStale || got.CurrentGitSHA != staleCodeSHA2 {
		t.Fatalf("refreshed stale code = %+v, want stale after freshness window", got)
	}
}

func assertStaleCodeEvidence(t *testing.T, got RuntimeStaleCodeEvidence, want string) {
	t.Helper()
	for _, evidence := range got.Evidence {
		if evidence == want {
			return
		}
	}
	t.Fatalf("stale code evidence = %#v, missing %q", got.Evidence, want)
}

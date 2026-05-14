package codingagents

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func newGuardOrSkip(t *testing.T, allowed []string) *WorkspaceGuard {
	t.Helper()
	g, err := NewWorkspaceGuard(allowed)
	if err != nil {
		t.Fatalf("NewWorkspaceGuard: %v", err)
	}
	return g
}

func TestWorkspaceGuard_AcceptsPathUnderAllowedRoot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	sub := filepath.Join(root, "project")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	g := newGuardOrSkip(t, []string{root})
	got, err := g.Resolve(sub)
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	want, _ := filepath.EvalSymlinks(sub)
	if got != want {
		t.Fatalf("Resolve = %q, want %q", got, want)
	}
}

func TestWorkspaceGuard_RefusesEmpty(t *testing.T) {
	t.Parallel()
	g := newGuardOrSkip(t, []string{t.TempDir()})
	for _, input := range []string{"", "   ", "\t"} {
		_, err := g.Resolve(input)
		if !errors.Is(err, ErrWorkspaceEmpty) {
			t.Fatalf("Resolve(%q) error = %v, want ErrWorkspaceEmpty", input, err)
		}
	}
}

func TestWorkspaceGuard_RefusesWhitespaceAmbiguous(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	g := newGuardOrSkip(t, []string{root})
	cases := []string{
		" " + root,
		root + " ",
		root + "  extra",
	}
	for _, input := range cases {
		_, err := g.Resolve(input)
		if !errors.Is(err, ErrWorkspaceAmbiguous) {
			t.Fatalf("Resolve(%q) error = %v, want ErrWorkspaceAmbiguous", input, err)
		}
	}
}

func TestWorkspaceGuard_RefusesOutsideAllowed(t *testing.T) {
	t.Parallel()
	allowed := t.TempDir()
	outside := t.TempDir()
	g := newGuardOrSkip(t, []string{allowed})
	_, err := g.Resolve(outside)
	if !errors.Is(err, ErrWorkspaceOutsideAllowed) {
		t.Fatalf("Resolve outside = %v, want ErrWorkspaceOutsideAllowed", err)
	}
}

func TestWorkspaceGuard_RefusesNonExistent(t *testing.T) {
	t.Parallel()
	allowed := t.TempDir()
	g := newGuardOrSkip(t, []string{allowed})
	_, err := g.Resolve(filepath.Join(allowed, "does-not-exist"))
	if err == nil {
		t.Fatalf("expected error for missing path, got nil")
	}
	// Must not collapse to a sentinel we'd treat as a successful guard.
	if errors.Is(err, ErrWorkspaceOutsideAllowed) || errors.Is(err, ErrWorkspaceDenied) {
		t.Fatalf("non-existent path leaked sentinel error: %v", err)
	}
}

func TestWorkspaceGuard_RefusesDeniedPaths(t *testing.T) {
	t.Parallel()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home dir: %v", err)
	}
	// Allowed root is the temp dir so it does not overlap the deny list.
	g := newGuardOrSkip(t, []string{t.TempDir()})
	denied := []string{"/", home}
	for _, candidate := range denied {
		_, err := g.Resolve(candidate)
		if !errors.Is(err, ErrWorkspaceDenied) && !errors.Is(err, ErrWorkspaceOutsideAllowed) {
			t.Fatalf("Resolve(%q) error = %v, want ErrWorkspaceDenied or ErrWorkspaceOutsideAllowed", candidate, err)
		}
	}
	// ~/.ssh may not exist on every CI host; only assert when it does.
	ssh := filepath.Join(home, ".ssh")
	if _, statErr := os.Stat(ssh); statErr == nil {
		_, err := g.Resolve(ssh)
		if !errors.Is(err, ErrWorkspaceDenied) {
			t.Fatalf("Resolve(~/.ssh) error = %v, want ErrWorkspaceDenied", err)
		}
	}
}

func TestNewWorkspaceGuard_RejectsAllowedRootOverlapDeny(t *testing.T) {
	t.Parallel()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home dir: %v", err)
	}
	if _, err := NewWorkspaceGuard([]string{home}); err == nil {
		t.Fatalf("expected error when allowed root equals denied $HOME")
	}
}

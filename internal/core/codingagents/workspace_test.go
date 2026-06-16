package codingagents

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
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

func TestWorkspaceGuard_AcceptsAllowedChildWithDotDotPrefix(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	sub := filepath.Join(root, "..project")
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

func TestWorkspaceGuard_UsesProfileWorkspaceScope(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	project1 := filepath.Join(root, "project1")
	project2 := filepath.Join(root, "project2")
	profile := filepath.Join(root, ".gormes", "profiles", "coder")
	sibling := filepath.Join(root, ".gormes", "profiles", "researcher")
	for _, dir := range []string{project1, project2, profile, sibling} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	scope, err := tools.NewProfileWorkspaceScope(tools.ProfileWorkspaceScopeOptions{
		ProjectRoots: []string{project1, project2},
		ProfileRoot:  profile,
		OperatorHome: root,
	})
	if err != nil {
		t.Fatalf("NewProfileWorkspaceScope: %v", err)
	}
	g, err := NewWorkspaceGuardFromProfileScope(scope)
	if err != nil {
		t.Fatalf("NewWorkspaceGuardFromProfileScope: %v", err)
	}

	if got, err := g.Resolve(project2); err != nil || got != project2 {
		t.Fatalf("Resolve(project2) = %q, %v; want %q", got, err, project2)
	}
	if _, err := g.Resolve(sibling); !errors.Is(err, ErrWorkspaceOutsideAllowed) {
		t.Fatalf("Resolve(sibling) error = %v, want ErrWorkspaceOutsideAllowed", err)
	}
	if _, err := g.Resolve(root); !errors.Is(err, ErrWorkspaceOutsideAllowed) {
		t.Fatalf("Resolve(root) error = %v, want ErrWorkspaceOutsideAllowed", err)
	}
}

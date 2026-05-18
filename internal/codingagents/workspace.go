package codingagents

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

// Sentinel errors so callers can present clear, voice-friendly messages and
// refuse ambiguous workspace inputs before dispatching a worker.
var (
	ErrWorkspaceEmpty          = errors.New("workspace identifier is empty")
	ErrWorkspaceAmbiguous      = errors.New("workspace identifier is ambiguous")
	ErrWorkspaceOutsideAllowed = errors.New("workspace is outside the allowed roots")
	ErrWorkspaceDenied         = errors.New("workspace is on the deny list")
)

// WorkspaceGuard enforces directory protection for coding-agent delegation.
// Callers must hand the guard a workspace identifier (typically a workspace
// ID or absolute path); the guard refuses anything ambiguous and confirms
// the resolved location is inside an allowed root and not under a denied
// path.
type WorkspaceGuard struct {
	// Scope is the shared Gormes profile workspace policy. When present it is
	// the first resolver used for delegation targets.
	Scope *tools.ProfileWorkspaceScope
	// AllowedRoots is the set of absolute, symlink-resolved roots inside
	// which workers may operate.
	AllowedRoots []string
	// DeniedPaths is the set of absolute, symlink-resolved paths workers
	// may never touch when targeted directly or by descent. It is seeded
	// from the per-user defaults at construct time.
	DeniedPaths []string
	// DeniedExact is the set of paths that are refused only when the
	// resolved workspace equals one of them. This lets containers like
	// $HOME be safe-by-default targets while still allowing project
	// checkouts inside them to be passed as allowed roots.
	DeniedExact []string
}

// NewWorkspaceGuard constructs a guard with the default deny list populated
// relative to the current user's $HOME. AllowedRoots is normalized to
// absolute symlink-resolved form. The constructor errors when an allowed
// root overlaps a subtree deny entry (for example, ~/.ssh) or matches the
// exact-deny set (such as $HOME itself).
func NewWorkspaceGuard(allowed []string) (*WorkspaceGuard, error) {
	denied, exact, err := defaultDenyList()
	if err != nil {
		return nil, err
	}
	normalized := make([]string, 0, len(allowed))
	for _, raw := range allowed {
		clean, err := canonicalize(raw)
		if err != nil {
			return nil, fmt.Errorf("allowed root %q: %w", raw, err)
		}
		for _, d := range denied {
			if pathIsUnderOrEqual(clean, d) {
				return nil, fmt.Errorf("%w: allowed root %q overlaps deny %q", ErrWorkspaceDenied, clean, d)
			}
		}
		for _, d := range exact {
			if clean == d {
				return nil, fmt.Errorf("%w: allowed root %q equals deny %q", ErrWorkspaceDenied, clean, d)
			}
		}
		normalized = append(normalized, clean)
	}
	return &WorkspaceGuard{AllowedRoots: normalized, DeniedPaths: denied, DeniedExact: exact}, nil
}

func NewWorkspaceGuardFromProfileScope(scope *tools.ProfileWorkspaceScope) (*WorkspaceGuard, error) {
	if scope == nil {
		return NewWorkspaceGuard(nil)
	}
	denied, exact, err := defaultDenyList()
	if err != nil {
		return nil, err
	}
	return &WorkspaceGuard{
		Scope:        scope,
		AllowedRoots: scope.ProjectRoots(),
		DeniedPaths:  denied,
		DeniedExact:  exact,
	}, nil
}

// Resolve validates the requested workspace and returns the absolute,
// symlink-resolved path the worker may use. It refuses empty or
// whitespace-ambiguous inputs, paths outside the allowed roots, and paths
// under any denied location.
func (g *WorkspaceGuard) Resolve(input string) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", ErrWorkspaceEmpty
	}
	if input != strings.TrimSpace(input) || strings.Contains(input, "  ") {
		return "", fmt.Errorf("%w: input has leading, trailing, or repeated whitespace", ErrWorkspaceAmbiguous)
	}
	if g.Scope != nil {
		decision := g.Scope.Resolve(input, g.Scope.DefaultRoot(), tools.ProfileWorkspaceAccessDelegate)
		if !decision.Allowed {
			return "", fmt.Errorf("%w: %s", ErrWorkspaceOutsideAllowed, decision.Message)
		}
		resolved := decision.Normalized
		if err := g.checkDenied(resolved, g.Scope.Configured()); err != nil {
			return "", err
		}
		return resolved, nil
	}
	resolved, err := canonicalize(input)
	if err != nil {
		return "", err
	}
	if resolved == string(filepath.Separator) {
		return "", fmt.Errorf("%w: filesystem root", ErrWorkspaceDenied)
	}
	if err := g.checkDenied(resolved, true); err != nil {
		return "", err
	}
	for _, root := range g.AllowedRoots {
		if pathIsUnderOrEqual(resolved, root) {
			return resolved, nil
		}
	}
	return "", fmt.Errorf("%w: %s", ErrWorkspaceOutsideAllowed, resolved)
}

func (g *WorkspaceGuard) checkDenied(resolved string, includeExact bool) error {
	if resolved == string(filepath.Separator) {
		return fmt.Errorf("%w: filesystem root", ErrWorkspaceDenied)
	}
	if includeExact {
		for _, d := range g.DeniedExact {
			if resolved == d {
				return fmt.Errorf("%w: %s", ErrWorkspaceDenied, resolved)
			}
		}
	}
	for _, d := range g.DeniedPaths {
		if pathIsUnderOrEqual(resolved, d) {
			return fmt.Errorf("%w: %s", ErrWorkspaceDenied, resolved)
		}
	}
	return nil
}

// defaultDenyList returns the per-user default deny entries split into
// subtree denies (resolved paths under any of these are refused) and exact
// denies (only the literal resolved path is refused). $HOME is an exact
// deny so legitimate project checkouts inside the home directory can be
// passed as allowed roots without colliding with broad subtree protection.
func defaultDenyList() (subtree []string, exact []string, err error) {
	home, herr := os.UserHomeDir()
	if herr != nil {
		return nil, nil, fmt.Errorf("resolve home directory: %w", herr)
	}
	relativeToHome := []string{".ssh", ".gormes", ".aws", ".gcloud", ".kube"}
	subtreeRaw := make([]string, 0, len(relativeToHome))
	for _, rel := range relativeToHome {
		subtreeRaw = append(subtreeRaw, filepath.Join(home, rel))
	}
	exactRaw := []string{home}
	subtree = normalizeForDeny(subtreeRaw)
	exact = normalizeForDeny(exactRaw)
	return subtree, exact, nil
}

// normalizeForDeny canonicalizes deny entries but tolerates missing dotdirs
// (such as ~/.aws on a fresh host) so the deny list still applies via the
// un-evaluated cleaned form.
func normalizeForDeny(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		c, err := canonicalize(p)
		if err != nil {
			out = append(out, filepath.Clean(p))
			continue
		}
		out = append(out, c)
	}
	return out
}

// canonicalize converts the input to absolute, symlink-resolved form. It
// requires that the path exists; non-existent inputs are refused so the
// guard never green-lights a typo'd workspace.
func canonicalize(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("absolute path for %q: %w", p, err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("resolve %q: %w", p, err)
	}
	return filepath.Clean(resolved), nil
}

// pathIsUnderOrEqual reports whether candidate equals base or sits inside it.
// It performs a clean comparison so trailing separators do not skew the
// result.
func pathIsUnderOrEqual(candidate, base string) bool {
	candidate = filepath.Clean(candidate)
	base = filepath.Clean(base)
	if candidate == base {
		return true
	}
	rel, err := filepath.Rel(base, candidate)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if strings.HasPrefix(rel, "..") {
		return false
	}
	return true
}

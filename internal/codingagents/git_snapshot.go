package codingagents

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ErrNotAGitRepo is returned when snapshot or diff calls are made against a
// directory that does not contain a git repository. Callers can branch on
// this to skip git-diff capture without surfacing a panic.
var ErrNotAGitRepo = errors.New("workspace is not a git repository")

// gitCmdTimeout bounds individual git invocations during snapshot/diff
// capture so a wedged repository cannot stall the worker.
const gitCmdTimeout = 5 * time.Second

// GitSnapshot captures repository state at one point in time so the
// coding-agent caller can describe what the worker changed.
type GitSnapshot struct {
	// Head is the resolved commit SHA at the time of snapshot.
	Head string
	// Branch is the current branch name, or "HEAD" if detached.
	Branch string
	// Dirty is true when the working tree had untracked or modified files.
	Dirty bool
	// Files lists `git status --porcelain` entries (workspace-relative).
	Files []string
}

// TakeSnapshot records HEAD, branch, dirty status, and the porcelain status
// list for the workspace. Returns ErrNotAGitRepo when the workspace is not
// a git checkout.
func TakeSnapshot(ctx context.Context, workspace string) (*GitSnapshot, error) {
	if err := requireGitRepo(ctx, workspace); err != nil {
		return nil, err
	}
	head, err := runGit(ctx, workspace, "rev-parse", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("rev-parse HEAD: %w", err)
	}
	branch, err := runGit(ctx, workspace, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("rev-parse --abbrev-ref HEAD: %w", err)
	}
	status, err := runGit(ctx, workspace, "status", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("status --porcelain: %w", err)
	}
	files := splitNonEmptyLines(status)
	return &GitSnapshot{
		Head:   strings.TrimSpace(head),
		Branch: strings.TrimSpace(branch),
		Dirty:  len(files) > 0,
		Files:  files,
	}, nil
}

// DiffBetween returns a unified diff describing what changed between two
// snapshots, plus the list of changed file paths. When the workspace is
// dirty after the worker run, working-tree changes are appended so callers
// see uncommitted edits as well.
func DiffBetween(ctx context.Context, workspace string, before, after *GitSnapshot) (string, []string, error) {
	if before == nil || after == nil {
		return "", nil, errors.New("before and after snapshots are required")
	}
	if err := requireGitRepo(ctx, workspace); err != nil {
		return "", nil, err
	}
	var diff strings.Builder
	if before.Head != after.Head {
		out, err := runGit(ctx, workspace, "diff", before.Head+".."+after.Head)
		if err != nil {
			return "", nil, fmt.Errorf("diff %s..%s: %w", before.Head, after.Head, err)
		}
		diff.WriteString(out)
	}
	if after.Dirty {
		wd, err := runGit(ctx, workspace, "diff")
		if err != nil {
			return "", nil, fmt.Errorf("diff working tree: %w", err)
		}
		diff.WriteString(wd)
	}
	files, err := changedFiles(ctx, workspace, before, after)
	if err != nil {
		return "", nil, err
	}
	return diff.String(), files, nil
}

// changedFiles enumerates the unique file paths that changed across the
// snapshot range plus any uncommitted working-tree edits.
func changedFiles(ctx context.Context, workspace string, before, after *GitSnapshot) ([]string, error) {
	seen := make(map[string]struct{})
	add := func(lines string) {
		for _, line := range splitNonEmptyLines(lines) {
			seen[line] = struct{}{}
		}
	}
	if before.Head != after.Head {
		names, err := runGit(ctx, workspace, "diff", "--name-only", before.Head+".."+after.Head)
		if err != nil {
			return nil, fmt.Errorf("diff --name-only %s..%s: %w", before.Head, after.Head, err)
		}
		add(names)
	}
	if after.Dirty {
		names, err := runGit(ctx, workspace, "diff", "--name-only")
		if err != nil {
			return nil, fmt.Errorf("diff --name-only working tree: %w", err)
		}
		add(names)
	}
	out := make([]string, 0, len(seen))
	for f := range seen {
		out = append(out, f)
	}
	return out, nil
}

// requireGitRepo verifies the directory is the root or interior of a git
// repository whose toplevel is the workspace itself (no walking up past the
// workspace into an ancestor repo). Returns ErrNotAGitRepo on failure so
// callers can branch on the typed error.
func requireGitRepo(ctx context.Context, workspace string) error {
	abs, err := filepath.Abs(workspace)
	if err != nil {
		return fmt.Errorf("abs workspace %q: %w", workspace, err)
	}
	out, err := runGit(ctx, abs, "rev-parse", "--show-toplevel")
	if err != nil {
		return ErrNotAGitRepo
	}
	top, err := filepath.EvalSymlinks(strings.TrimSpace(out))
	if err != nil {
		return ErrNotAGitRepo
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return ErrNotAGitRepo
	}
	if filepath.Clean(top) != filepath.Clean(resolved) {
		// The closest repo is in an ancestor, not the workspace itself.
		return ErrNotAGitRepo
	}
	return nil
}

// runGit executes a git subcommand against the workspace and returns stdout.
// GIT_CEILING_DIRECTORIES is set to the workspace's parent so git cannot
// walk above the workspace and silently bind to an ancestor repository.
func runGit(ctx context.Context, workspace string, args ...string) (string, error) {
	runCtx, cancel := context.WithTimeout(ctx, gitCmdTimeout)
	defer cancel()
	full := append([]string{"-C", workspace}, args...)
	cmd := exec.CommandContext(runCtx, "git", full...)
	cmd.Env = append(os.Environ(), "GIT_CEILING_DIRECTORIES="+filepath.Dir(workspace))
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// splitNonEmptyLines splits text on newlines and trims blank entries.
func splitNonEmptyLines(text string) []string {
	raw := strings.Split(text, "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		trimmed := strings.TrimRight(line, "\r")
		if strings.TrimSpace(trimmed) == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

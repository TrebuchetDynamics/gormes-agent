package autoloop

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func repoHasGit(repoRoot string) bool {
	if repoRoot == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(repoRoot, ".git"))
	return err == nil
}

func gitCurrentBranch(repoRoot string) (string, error) {
	cmd := exec.Command("git", "-C", repoRoot, "branch", "--show-current")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git branch --show-current: %w", err)
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" {
		return "", fmt.Errorf("git branch --show-current returned empty (detached HEAD?)")
	}
	return branch, nil
}

func gitCreateWorkerBranch(repoRoot, branch string) error {
	cmd := exec.Command("git", "-C", repoRoot, "switch", "-c", branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git switch -c %s: %w: %s", branch, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func gitRestoreBranch(repoRoot, branch string) error {
	if branch == "" {
		return nil
	}
	cmd := exec.Command("git", "-C", repoRoot, "switch", branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git switch %s: %w: %s", branch, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func restoreBranchIfClean(repoRoot string, branch string) {
	if ensureWorktreeClean(repoRoot) != nil {
		return
	}
	_ = gitRestoreBranch(repoRoot, branch)
}

func gitHeadSha(repoRoot string) (string, error) {
	cmd := exec.Command("git", "-C", repoRoot, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func gitWorkingTreeDirty(repoRoot string) (bool, error) {
	cmd := exec.Command("git", "-C", repoRoot, "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git status --porcelain: %w", err)
	}
	return strings.TrimSpace(string(out)) != "", nil
}

func gitCommitAll(repoRoot, message string) error {
	if out, err := exec.Command("git", "-C", repoRoot, "add", "-A").CombinedOutput(); err != nil {
		return fmt.Errorf("git add -A: %w: %s", err, strings.TrimSpace(string(out)))
	}
	cmd := exec.Command("git", "-C", repoRoot, "commit", "-m", message)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=gormes-autoloop",
		"GIT_AUTHOR_EMAIL=autoloop@gormes.local",
		"GIT_COMMITTER_NAME=gormes-autoloop",
		"GIT_COMMITTER_EMAIL=autoloop@gormes.local",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func gitCleanupAfterFailure(repoRoot string) {
	_ = exec.Command("git", "-C", repoRoot, "merge", "--abort").Run()
	_ = exec.Command("git", "-C", repoRoot, "reset", "--hard").Run()
	_ = exec.Command("git", "-C", repoRoot, "clean", "-fd").Run()
}

func sanitizeBranchSegment(value string) string {
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + 32)
		case r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	out := b.String()
	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}
	out = strings.Trim(out, "-_.")
	if len(out) > 60 {
		out = strings.TrimRight(out[:60], "-_.")
	}
	if out == "" {
		return "task"
	}
	return out
}

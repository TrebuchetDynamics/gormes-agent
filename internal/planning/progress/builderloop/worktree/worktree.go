package worktree

import (
	"fmt"
	"path/filepath"
	"strings"
)

type CandidateRef struct {
	PhaseID    string
	SubphaseID string
	ItemName   string
}

type ConfigRef struct {
	RepoRoot string
	RunRoot  string
}

func WorkerBranchName(runID string, workerID int, candidate CandidateRef) string {
	slug := SanitizeBranchSegment(candidate.PhaseID + "-" + candidate.SubphaseID + "-" + candidate.ItemName)
	return fmt.Sprintf("builder-loop/%s/w%d/%s", runID, workerID, slug)
}

func WorkerWorktreePath(cfg ConfigRef, runID string, workerID int) string {
	runRoot := cfg.RunRoot
	if runRoot == "" {
		runRoot = filepath.Join(cfg.RepoRoot, ".codex", "builder-loop")
	}
	return filepath.Join(runRoot, "worktrees", runID, fmt.Sprintf("w%d", workerID))
}

func WorkerRepoRoot(workerRoot string, repoSubdir string) string {
	if repoSubdir == "" || repoSubdir == "." {
		return workerRoot
	}

	return filepath.Join(workerRoot, repoSubdir)
}

func SanitizeBranchSegment(value string) string {
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

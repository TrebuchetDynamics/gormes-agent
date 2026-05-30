package builderloop

import "github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/builderloop/worktree"

func WorkerBranchName(runID string, workerID int, candidate Candidate) string {
	return worktree.WorkerBranchName(runID, workerID, worktree.CandidateRef{
		PhaseID:    candidate.PhaseID,
		SubphaseID: candidate.SubphaseID,
		ItemName:   candidate.ItemName,
	})
}

func WorkerWorktreePath(cfg Config, runID string, workerID int) string {
	return worktree.WorkerWorktreePath(worktree.ConfigRef{
		RepoRoot: cfg.RepoRoot,
		RunRoot:  cfg.RunRoot,
	}, runID, workerID)
}

func WorkerRepoRoot(workerRoot string, repoSubdir string) string {
	return worktree.WorkerRepoRoot(workerRoot, repoSubdir)
}

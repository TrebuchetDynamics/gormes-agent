package plannerloop

import "github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/plannerloop/backendcmd"

type BackendFailure = backendcmd.BackendFailure

func buildBackendCommandWithRepoRoot(backend, mode, repoRoot string) ([]string, error) {
	return backendcmd.BuildCommandWithRepoRoot(backend, mode, repoRoot)
}

func sandboxForMode(mode string) (string, error) {
	return backendcmd.SandboxForMode(mode)
}

func classifyBackendFailure(err error, stdout, stderr string) BackendFailure {
	return backendcmd.ClassifyFailure(err, stdout, stderr)
}

func backendFailureDetail(err error, stdout, stderr string) string {
	return backendcmd.FailureDetail(err, stdout, stderr)
}

func backendWasKilled(text string) bool {
	return backendcmd.WasKilled(text)
}

func backendHitUsageLimit(text string) bool {
	return backendcmd.HitUsageLimit(text)
}

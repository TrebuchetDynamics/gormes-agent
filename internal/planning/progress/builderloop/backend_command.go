package builderloop

import "github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/builderloop/backend"

func BuildBackendCommand(backendName, mode string) ([]string, error) {
	return backend.BuildCommand(backendName, mode)
}

func BuildBackendCommandWithRepoRoot(backendName, mode, repoRoot string) ([]string, error) {
	return backend.BuildCommandWithRepoRoot(backendName, mode, repoRoot)
}

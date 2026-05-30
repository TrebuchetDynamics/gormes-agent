package builderloop

import "github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/builderloop/backend"

type BackendFailure = backend.BackendFailure

func ClassifyBackendFailure(err error, stdout, stderr string) BackendFailure {
	return backend.ClassifyBackendFailure(err, stdout, stderr)
}

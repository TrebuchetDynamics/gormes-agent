package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/termux"

func isTermuxToolEnv(env func(string) string) bool {
	return termux.IsEnvironment(env)
}

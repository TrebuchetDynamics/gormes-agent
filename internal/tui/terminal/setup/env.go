package setup

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/envmap"

func envValue(env map[string]string, key string) string {
	return envmap.Value(env, key)
}

func envHas(env map[string]string, key string) bool {
	return envmap.Has(env, key)
}

func isRemoteTerminal(env map[string]string) bool {
	return envValue(env, "SSH_CONNECTION") != "" || envValue(env, "SSH_TTY") != ""
}

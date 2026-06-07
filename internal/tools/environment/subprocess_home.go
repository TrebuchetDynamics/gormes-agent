package environment

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/environment/subprocess"

// SubprocessHomeResolver returns the profile-local HOME that should be used by
// local shell subprocesses. It is injected from config to keep tools free of
// config package dependencies and to make the behavior directly testable.
type SubprocessHomeResolver = subprocess.HomeResolver

// EnvWithSubprocessHome overlays the resolved profile-local HOME onto env.
func EnvWithSubprocessHome(env []string, resolve SubprocessHomeResolver) []string {
	return subprocess.EnvWithHome(env, resolve)
}

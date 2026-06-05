package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/environment"

// SubprocessHomeResolver returns the profile-local HOME that should be used
// by local shell subprocesses. It is injected from config to keep tools free of
// config package dependencies and to make the behavior directly testable.
type SubprocessHomeResolver = environment.SubprocessHomeResolver

func envWithSubprocessHome(env []string, resolve SubprocessHomeResolver) []string {
	return environment.EnvWithSubprocessHome(env, resolve)
}

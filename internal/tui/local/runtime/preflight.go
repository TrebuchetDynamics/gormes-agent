package runtime

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/doctor"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/local/runtime/preflight"
)

type StartupPreflightOptions = preflight.StartupPreflightOptions

func RunNativeStartupPreflight(ctx context.Context, opts StartupPreflightOptions) doctor.CheckResult {
	return preflight.RunNativeStartupPreflight(ctx, opts)
}

func DoctorStatus() doctor.CheckResult {
	return preflight.DoctorStatus()
}

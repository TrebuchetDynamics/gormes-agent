package local

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/doctor"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/local/runtime"
)

type StartupPreflightOptions = runtime.StartupPreflightOptions

func RunNativeStartupPreflight(ctx context.Context, opts StartupPreflightOptions) doctor.CheckResult {
	return runtime.RunNativeStartupPreflight(ctx, opts)
}

func DoctorStatus() doctor.CheckResult {
	return runtime.DoctorStatus()
}

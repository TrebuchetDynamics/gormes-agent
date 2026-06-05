package gormescli

import (
	"context"

	appdoctorbrowser "github.com/TrebuchetDynamics/gormes-agent/internal/app/doctorbrowser"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/doctor"
)

type BrowserRuntimeDoctorDeps struct {
	LookPath func(string) (string, error)
	Getenv   func(string) string
	ProbeCDP func(context.Context, string) error
	Offline  bool
}

func DoctorBrowserRuntimeStatus() doctor.CheckResult {
	return appdoctorbrowser.RuntimeStatus()
}

func DoctorBrowserRuntimeStatusWithDeps(deps BrowserRuntimeDoctorDeps) doctor.CheckResult {
	return appdoctorbrowser.RuntimeStatusWithDeps(appdoctorbrowser.RuntimeDoctorDeps{
		LookPath: deps.LookPath,
		Getenv:   deps.Getenv,
		ProbeCDP: deps.ProbeCDP,
		Offline:  deps.Offline,
	})
}

func DoctorBrowserChromeLaunchCommand(chromePath string) string {
	return appdoctorbrowser.ChromeLaunchCommand(chromePath)
}

func DoctorBrowserChromeCDPVersionURL(endpoint string) (string, error) {
	return appdoctorbrowser.ChromeCDPVersionURL(endpoint)
}

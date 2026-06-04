package main

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/doctor"
)

const defaultChromeCDPEndpoint = "http://127.0.0.1:9222"

var chromeExecutableCandidates = []string{
	"google-chrome",
	"google-chrome-stable",
	"chromium",
	"chromium-browser",
	"chrome",
	"brave-browser",
	"microsoft-edge",
	"microsoft-edge-stable",
}

type browserRuntimeDoctorDeps struct {
	lookPath func(string) (string, error)
	getenv   func(string) string
	probeCDP func(context.Context, string) error
	offline  bool
}

func doctorBrowserRuntimeStatus() doctor.CheckResult {
	return gormescli.DoctorBrowserRuntimeStatus()
}

func doctorBrowserRuntimeStatusWithDeps(deps browserRuntimeDoctorDeps) doctor.CheckResult {
	return gormescli.DoctorBrowserRuntimeStatusWithDeps(gormescli.BrowserRuntimeDoctorDeps{
		LookPath: deps.lookPath,
		Getenv:   deps.getenv,
		ProbeCDP: deps.probeCDP,
		Offline:  deps.offline,
	})
}

func chromeLaunchCommand(chromePath string) string {
	return gormescli.DoctorBrowserChromeLaunchCommand(chromePath)
}

func chromeCDPVersionURL(endpoint string) (string, error) {
	return gormescli.DoctorBrowserChromeCDPVersionURL(endpoint)
}

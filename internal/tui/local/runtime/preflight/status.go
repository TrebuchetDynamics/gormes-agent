package preflight

import (
	"context"
	"os"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/doctor"
)

type StartupPreflightOptions struct {
	WorkDir string
}

func RunNativeStartupPreflight(_ context.Context, opts StartupPreflightOptions) doctor.CheckResult {
	if opts.WorkDir == "" {
		if wd, err := os.Getwd(); err == nil {
			opts.WorkDir = wd
		}
	}
	return DoctorStatus()
}

func DoctorStatus() doctor.CheckResult {
	return doctor.CheckResult{
		Name:    "Native TUI",
		Status:  doctor.StatusPass,
		Summary: "available: Go-native Bubble Tea TUI compiled into gormes",
		Items: []doctor.ItemInfo{
			{
				Name:   "runtime",
				Status: doctor.StatusPass,
				Note:   "local startup uses the compiled Go Bubble Tea shell",
			},
			{
				Name:   "offline",
				Status: doctor.StatusPass,
				Note:   "offline mode keeps the same native TUI path without a JavaScript bundle build step",
			},
			{
				Name:   "remote",
				Status: doctor.StatusPass,
				Note:   "remote streaming via --remote <url> consumes native Gormes turn events over SSE or websocket attach via GORMES_TUI_GATEWAY_URL/HERMES_TUI_GATEWAY_URL; without a remote URL the local Bubble Tea path remains the runtime",
			},
		},
	}
}

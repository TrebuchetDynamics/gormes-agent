package main

import (
	"fmt"

	"github.com/TrebuchetDynamics/gormes-agent/internal/doctor"
)

const gatewayTermuxLifecycleGuidanceLine = "Termux gateway: foreground/tmux lifecycle; run `gormes gateway` inside tmux; termux-wake-lock and Android battery settings are best-effort only, and Android may still stop background processes."

func gatewayTermuxDetected() bool {
	return doctor.IsTermuxEnvironment(nil)
}

func gatewayTermuxLifecycleGuidanceError(action string) error {
	return fmt.Errorf("gateway: %s uses the Termux foreground/tmux lifecycle; run `gormes gateway` inside tmux; termux-wake-lock and Android battery settings are best-effort only, and Android may still stop background processes; use `gormes gateway status` and `gormes gateway stop` for runtime control", action)
}

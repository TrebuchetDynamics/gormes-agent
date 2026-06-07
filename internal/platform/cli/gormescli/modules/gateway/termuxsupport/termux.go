package termuxsupport

import (
	"context"
	"fmt"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/doctor"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

const LifecycleGuidanceLine = "Termux gateway: foreground/tmux lifecycle; run `gormes gateway` inside tmux; termux-wake-lock and Android battery settings are best-effort only, and Android may still stop background processes."

func Detected() bool {
	return doctor.IsTermuxEnvironment(nil)
}

func LifecycleGuidanceError(action string) error {
	return fmt.Errorf("gateway: %s uses the Termux foreground/tmux lifecycle; run `gormes gateway` inside tmux; termux-wake-lock and Android battery settings are best-effort only, and Android may still stop background processes; use `gormes gateway status` and `gormes gateway stop` for runtime control", action)
}

func NotificationStatusLine() string {
	if !Detected() {
		return ""
	}
	result := tools.TermuxNotificationSender{}.Status(context.Background())
	switch result.Status {
	case tools.TermuxNotificationStatusAvailable:
		return "Termux notification: available command=termux-notification\n"
	case tools.TermuxNotificationStatusUnavailable:
		return fmt.Sprintf("Termux notification: %s message=%q\n", result.Status, result.Message)
	default:
		return ""
	}
}

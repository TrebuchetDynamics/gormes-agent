package gateway

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/gateway/termuxsupport"
)

const TermuxLifecycleGuidanceLine = termuxsupport.LifecycleGuidanceLine

func TermuxDetected() bool {
	return termuxsupport.Detected()
}

func TermuxLifecycleGuidanceError(action string) error {
	return termuxsupport.LifecycleGuidanceError(action)
}

func TermuxNotificationStatusLine() string {
	return termuxsupport.NotificationStatusLine()
}

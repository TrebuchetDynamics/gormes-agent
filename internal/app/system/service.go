package system

import (
	"fmt"
	"strings"

	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func ParseEventMode(raw string) (toolspkg.SystemEventMode, error) {
	switch toolspkg.SystemEventMode(strings.TrimSpace(raw)) {
	case "", toolspkg.SystemEventModeNextHeartbeat:
		return toolspkg.SystemEventModeNextHeartbeat, nil
	case toolspkg.SystemEventModeNow:
		return toolspkg.SystemEventModeNow, nil
	default:
		return "", fmt.Errorf("system event: --mode must be now or next-heartbeat")
	}
}

func FirstDegradedMessage(items []toolspkg.SystemDegradedStatus) string {
	if len(items) == 0 {
		return "system_unavailable"
	}
	item := items[0]
	if item.Message != "" {
		return item.Message
	}
	if item.Reason != "" {
		return item.Reason
	}
	return item.Code
}

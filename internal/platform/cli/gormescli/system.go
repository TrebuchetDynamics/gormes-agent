package gormescli

import (
	appsystem "github.com/TrebuchetDynamics/gormes-agent/internal/app/system"
	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func ParseSystemEventMode(raw string) (toolspkg.SystemEventMode, error) {
	return appsystem.ParseEventMode(raw)
}

func FirstSystemDegradedMessage(items []toolspkg.SystemDegradedStatus) string {
	return appsystem.FirstDegradedMessage(items)
}

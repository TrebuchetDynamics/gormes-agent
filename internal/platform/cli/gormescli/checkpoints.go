package gormescli

import (
	"time"

	appcheckpoints "github.com/TrebuchetDynamics/gormes-agent/internal/app/checkpoints"
)

func FormatCheckpointBytes(n int64) string {
	return appcheckpoints.FormatBytes(n)
}

func CheckpointAgo(now, t time.Time) time.Duration {
	return appcheckpoints.Ago(now, t)
}

func FormatCheckpointAge(d time.Duration) string {
	return appcheckpoints.FormatAge(d)
}

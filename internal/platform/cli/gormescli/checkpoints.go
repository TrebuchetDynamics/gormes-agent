package gormescli

import (
	"time"

	"github.com/spf13/cobra"

	appcheckpoints "github.com/TrebuchetDynamics/gormes-agent/internal/app/checkpoints"
)

type CheckpointsBuildProvenance = appcheckpoints.BuildProvenance

func NewCheckpointsCommand(build func() BuildProvenance) *cobra.Command {
	appcheckpoints.BuildProvenanceFunc = func() appcheckpoints.BuildProvenance {
		if build == nil {
			return appcheckpoints.BuildProvenance{}
		}
		provenance := build()
		return appcheckpoints.BuildProvenance{Version: provenance.Version, GitCommit: provenance.GitCommit}
	}
	return appcheckpoints.NewCommand()
}

func FormatCheckpointBytes(n int64) string {
	return appcheckpoints.FormatBytes(n)
}

func CheckpointAgo(now, t time.Time) time.Duration {
	return appcheckpoints.Ago(now, t)
}

func FormatCheckpointAge(d time.Duration) string {
	return appcheckpoints.FormatAge(d)
}

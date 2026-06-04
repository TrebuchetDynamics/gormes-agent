package gormescli

import (
	"github.com/spf13/cobra"

	appsystem "github.com/TrebuchetDynamics/gormes-agent/internal/app/system"
	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

type SystemBuildProvenance = appsystem.BuildProvenance

func NewSystemCommand(build func() BuildProvenance) *cobra.Command {
	appsystem.BuildProvenanceFunc = func() appsystem.BuildProvenance {
		if build == nil {
			return appsystem.BuildProvenance{}
		}
		provenance := build()
		return appsystem.BuildProvenance{Version: provenance.Version, GitCommit: provenance.GitCommit}
	}
	return appsystem.NewCommand()
}

func DefaultSystemEventsManager() toolspkg.SystemEventsManager { return appsystem.DefaultManager() }

func ParseSystemEventMode(raw string) (toolspkg.SystemEventMode, error) {
	return appsystem.ParseEventMode(raw)
}

func FirstSystemDegradedMessage(items []toolspkg.SystemDegradedStatus) string {
	return appsystem.FirstDegradedMessage(items)
}

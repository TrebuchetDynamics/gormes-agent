package main

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func newSystemCommand() *cobra.Command {
	return gormescli.NewSystemCommand(func() gormescli.BuildProvenance {
		provenance := newBuildProvenance()
		return gormescli.BuildProvenance{Version: provenance.Version, GitCommit: provenance.GitCommit}
	})
}

func cliSystemEventsManager() toolspkg.SystemEventsManager {
	return gormescli.DefaultSystemEventsManager()
}

func parseSystemEventMode(raw string) (toolspkg.SystemEventMode, error) {
	return gormescli.ParseSystemEventMode(raw)
}

func firstSystemDegradedMessage(items []toolspkg.SystemDegradedStatus) string {
	return gormescli.FirstSystemDegradedMessage(items)
}

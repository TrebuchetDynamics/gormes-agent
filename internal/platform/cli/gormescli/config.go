package gormescli

import (
	"github.com/spf13/cobra"

	appconfig "github.com/TrebuchetDynamics/gormes-agent/internal/app/config"
)

type ConfigBuildProvenance = appconfig.BuildProvenance

type ConfigEditorRunner = appconfig.EditorRunner

type ConfigOSEditorRunner = appconfig.OSEditorRunner

func NewConfigCommand(build func() BuildProvenance) *cobra.Command {
	appconfig.BuildProvenanceFunc = func() appconfig.BuildProvenance {
		if build == nil {
			return appconfig.BuildProvenance{}
		}
		provenance := build()
		return appconfig.BuildProvenance{Version: provenance.Version, GitCommit: provenance.GitCommit}
	}
	return appconfig.NewCommand()
}

func WithConfigEditorRunner(t interface{ Cleanup(func()) }, runner ConfigEditorRunner) {
	appconfig.WithEditorRunner(t, runner)
}

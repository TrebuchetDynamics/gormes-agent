package main

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

type editorRunner = gormescli.ConfigEditorRunner
type osEditorRunner = gormescli.ConfigOSEditorRunner

func withConfigEditorRunner(t interface{ Cleanup(func()) }, runner editorRunner) {
	gormescli.WithConfigEditorRunner(t, runner)
}

func newConfigCommand() *cobra.Command {
	return gormescli.NewConfigCommand(func() gormescli.BuildProvenance {
		build := newBuildProvenance()
		return gormescli.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
	})
}

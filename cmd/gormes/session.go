package main

import (
	"database/sql"

	"github.com/spf13/cobra"

	sessionpkg "github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func newSessionCommand() *cobra.Command {
	return gormescli.NewSessionCommand(gormescli.SessionCommandOptions{
		Build: func() gormescli.SessionBuildProvenance {
			build := newBuildProvenance()
			return gormescli.SessionBuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
		},
		UnavailableCommand: func(spec gormescli.SessionUnavailableCommandSpec) *cobra.Command {
			return newHermesUnavailableCommand(hermesUnavailableCommandSpec{Use: spec.Use, Short: spec.Short, Row: spec.Row})
		},
	})
}

func resolveContinueSessionFlag(raw string) (string, error) {
	return gormescli.ResolveContinueSessionFlag(raw)
}

func coalesceSessionNameArgs(argv []string) []string {
	return gormescli.CoalesceSessionNameArgs(argv)
}

func newTUISaveExportFunc() tui.SessionExportFunc {
	return gormescli.NewTUISaveExportFunc()
}

func openSessionDirectoryDB() (*sql.DB, error) {
	return gormescli.OpenSessionDirectoryDB()
}

func applySessionMirrorSources(entries []sessionpkg.DirectoryEntry, mirrorPath string) []sessionpkg.DirectoryEntry {
	return gormescli.ApplySessionMirrorSources(entries, mirrorPath)
}

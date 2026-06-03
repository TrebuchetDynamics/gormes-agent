package gormescli

import (
	"database/sql"

	"github.com/spf13/cobra"

	appsession "github.com/TrebuchetDynamics/gormes-agent/internal/app/session"
	sessionpkg "github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/sessions"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

type SessionBuildProvenance = appsession.BuildProvenance

type SessionUnavailableCommandSpec = sessions.UnavailableCommandSpec

type SessionCommandOptions struct {
	Build              func() SessionBuildProvenance
	UnavailableCommand func(SessionUnavailableCommandSpec) *cobra.Command
}

func NewSessionCommand(options SessionCommandOptions) *cobra.Command {
	return appsession.NewCommand(appsession.CommandOptions{
		Build: options.Build,
		UnavailableCommand: func(spec sessions.UnavailableCommandSpec) *cobra.Command {
			if options.UnavailableCommand == nil {
				return nil
			}
			return options.UnavailableCommand(spec)
		},
	})
}

func ResolveContinueSessionFlag(raw string) (string, error) {
	return appsession.ResolveContinueSessionFlag(raw)
}

func CoalesceSessionNameArgs(argv []string) []string {
	return appsession.CoalesceSessionNameArgs(argv)
}

func NewTUISaveExportFunc() tui.SessionExportFunc {
	return appsession.NewTUISaveExportFunc()
}

func OpenSessionDirectoryDB() (*sql.DB, error) {
	return appsession.OpenSessionDirectoryDB()
}

func ApplySessionMirrorSources(entries []sessionpkg.DirectoryEntry, mirrorPath string) []sessionpkg.DirectoryEntry {
	return appsession.ApplySessionMirrorSources(entries, mirrorPath)
}

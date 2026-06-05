package gormescli

import (
	"context"

	"github.com/spf13/cobra"

	appstatus "github.com/TrebuchetDynamics/gormes-agent/internal/app/status"
	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

type StatusSnapshotFunc = appstatus.SnapshotFunc

type StatusCommandOptions struct {
	BuildProvenance func() BuildProvenance
	SystemSnapshot  func(context.Context) (toolspkg.SystemEventsSnapshot, error)
	AuditPath       func() string
}

func NewStatusCommand(options StatusCommandOptions) *cobra.Command {
	return appstatus.NewCommand(appstatus.Options{
		BuildProvenance: func() appstatus.BuildProvenance {
			if options.BuildProvenance == nil {
				return appstatus.BuildProvenance{}
			}
			provenance := options.BuildProvenance()
			return appstatus.BuildProvenance{Version: provenance.Version, GitCommit: provenance.GitCommit}
		},
		SystemSnapshot: appstatus.SnapshotFunc(options.SystemSnapshot),
		AuditPath:      options.AuditPath,
	})
}

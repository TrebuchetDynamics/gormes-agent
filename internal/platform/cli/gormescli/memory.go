package gormescli

import (
	"database/sql"

	goncho "github.com/TrebuchetDynamics/goncho/service"
	"github.com/spf13/cobra"

	appmemory "github.com/TrebuchetDynamics/gormes-agent/internal/app/memory"
)

type MemoryCommandOptions = appmemory.Options

type MemoryBuildProvenance = appmemory.BuildProvenance

type MemoryStatusReportJSON = appmemory.StatusReportJSON

type MemoryExtractorJSON = appmemory.ExtractorJSON

func NewMemoryStatusCommand(opts MemoryCommandOptions) *cobra.Command {
	return appmemory.NewStatusCommand(opts)
}

func FormatMemoryDreamQueueEvidence(status goncho.DreamQueueStatus) string {
	return appmemory.FormatDreamQueueEvidence(status)
}

func MemoryOptions(build func() BuildProvenance, openDB func(path string) (*sql.DB, error)) MemoryCommandOptions {
	return MemoryCommandOptions{
		BuildProvenance: func() appmemory.BuildProvenance {
			if build == nil {
				return appmemory.BuildProvenance{Version: "unknown", GitCommit: "unknown"}
			}
			got := build()
			return appmemory.BuildProvenance{Version: got.Version, GitCommit: got.GitCommit}
		},
		OpenDB: openDB,
	}
}

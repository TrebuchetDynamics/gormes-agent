package gormescli

import (
	"github.com/spf13/cobra"

	apprestore "github.com/TrebuchetDynamics/gormes-agent/internal/app/restore"
)

type RestoreCommandSeams = apprestore.Seams

func NewRestoreCommand(build func() BuildProvenance, jsonInputError func(*cobra.Command, string, string) error) *cobra.Command {
	return apprestore.NewCommand(restoreOptions(build, jsonInputError))
}

func NewRestoreCommandWithSeams(seams RestoreCommandSeams, build func() BuildProvenance, jsonInputError func(*cobra.Command, string, string) error) *cobra.Command {
	return apprestore.NewCommandWithSeams(seams, restoreOptions(build, jsonInputError))
}

func restoreOptions(build func() BuildProvenance, jsonInputError func(*cobra.Command, string, string) error) apprestore.Options {
	return apprestore.Options{
		BuildProvenance: func() apprestore.BuildProvenance {
			if build == nil {
				return apprestore.BuildProvenance{}
			}
			provenance := build()
			return apprestore.BuildProvenance{Version: provenance.Version, GitCommit: provenance.GitCommit}
		},
		JSONInputError: jsonInputError,
	}
}

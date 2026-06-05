package gormescli

import "github.com/spf13/cobra"

type restoreCommandSeams = RestoreCommandSeams

func newRestoreCommandWithSeams(seams restoreCommandSeams) *cobra.Command {
	return NewRestoreCommand(RestoreCommandOptions{
		BuildProvenance: func() BuildProvenance {
			return BuildProvenance{Version: Version, GitCommit: "test-git"}
		},
		Seams: seams,
	})
}

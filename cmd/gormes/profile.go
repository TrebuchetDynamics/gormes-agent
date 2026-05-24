package main

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli/gormescli"
	profilemodule "github.com/TrebuchetDynamics/gormes-agent/internal/cli/gormescli/modules/profiles"
)

type profileCommandSeams = profilemodule.Seams

func newProfileCommand() *cobra.Command {
	seams := defaultProfileCommandSeams()
	cmd := profilemodule.NewCommandWithSeams(seams, profileCommandOptions())
	cmd.AddCommand(gormescli.NewProfileSeedCommand(profileSeedSeamsFromProfileSeams(seams)))
	return cmd
}

func newProfileCommandWithSeams(seams profileCommandSeams) *cobra.Command {
	cmd := profilemodule.NewCommandWithSeams(seams, profileCommandOptions())
	cmd.AddCommand(gormescli.NewProfileSeedCommand(profileSeedSeamsFromProfileSeams(seams)))
	return cmd
}

func defaultProfileCommandSeams() profileCommandSeams {
	return profilemodule.DefaultSeams()
}

func defaultListKnownProfiles() ([]string, error) {
	return profilemodule.DefaultListKnownProfiles()
}

func profileCommandOptions() profilemodule.Options {
	return profilemodule.Options{
		BuildProvenance: func() gormescli.BuildProvenance {
			build := newBuildProvenance()
			return gormescli.BuildProvenance{
				Version:   build.Version,
				GitCommit: build.GitCommit,
			}
		},
	}
}

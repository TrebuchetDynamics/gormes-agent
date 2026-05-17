package main

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/app/gormescli"
	profilemodule "github.com/TrebuchetDynamics/gormes-agent/internal/app/gormescli/modules/profiles"
)

type profileCommandSeams = profilemodule.Seams

func newProfileCommand() *cobra.Command {
	return profilemodule.NewCommand(profileCommandOptions())
}

func newProfileCommandWithSeams(seams profileCommandSeams) *cobra.Command {
	return profilemodule.NewCommandWithSeams(seams, profileCommandOptions())
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

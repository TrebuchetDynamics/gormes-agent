package main

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/profileapp"
)

type profileCommandSeams = profileapp.CommandSeams

func newProfileCommand() *cobra.Command {
	return profileapp.NewCommand(profileBuildProvenance)
}

func newProfileCommandWithSeams(seams profileCommandSeams) *cobra.Command {
	return profileapp.NewCommandWithSeams(seams, profileCommandOptions())
}

func defaultProfileCommandSeams() profileCommandSeams {
	return profileapp.DefaultSeams()
}

func defaultListKnownProfiles() ([]string, error) {
	return profileapp.DefaultListKnownProfiles()
}

func profileCommandOptions() profileapp.CommandOptions {
	return profileapp.CommandOptions{BuildProvenance: profileBuildProvenance}
}

func profileBuildProvenance() gormescli.BuildProvenance {
	build := newBuildProvenance()
	return gormescli.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
}

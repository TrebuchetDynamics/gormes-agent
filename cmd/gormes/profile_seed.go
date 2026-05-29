package main

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"

func profileSeedSeamsFromProfileSeams(seams profileCommandSeams) gormescli.ProfileSeedSeams {
	return gormescli.ProfileSeedSeams{
		CreateProfile: seams.CreateProfile,
		BuildProvenance: func() gormescli.BuildProvenance {
			build := newBuildProvenance()
			return gormescli.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
		},
	}
}

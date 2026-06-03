package main

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/profileapp"
)

func profileSeedSeamsFromProfileSeams(seams profileCommandSeams) gormescli.ProfileSeedSeams {
	return profileapp.ProfileSeedSeamsFromProfileSeams(seams, profileBuildProvenance)
}

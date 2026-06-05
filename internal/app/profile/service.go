package profile

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/commandruntime"
	profilemodule "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/profiles"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/profileseedruntime"
)

type Seams = profilemodule.Seams

type Options = profilemodule.Options

type BuildProvenance = commandruntime.BuildProvenance

type ProfileSeedSeams = profileseedruntime.ProfileSeedSeams

func NewCommand(build func() BuildProvenance) *cobra.Command {
	seams := DefaultSeams()
	return NewCommandWithSeams(seams, Options{BuildProvenance: build})
}

func NewCommandWithSeams(seams Seams, opts Options) *cobra.Command {
	cmd := profilemodule.NewCommandWithSeams(seams, opts)
	cmd.AddCommand(profileseedruntime.NewProfileSeedCommand(ProfileSeedSeamsFromProfileSeams(seams, opts.BuildProvenance)))
	return cmd
}

func DefaultSeams() Seams {
	return profilemodule.DefaultSeams()
}

func DefaultListKnownProfiles() ([]string, error) {
	return profilemodule.DefaultListKnownProfiles()
}

func ProfileSeedSeamsFromProfileSeams(seams Seams, build func() BuildProvenance) ProfileSeedSeams {
	return ProfileSeedSeams{
		CreateProfile: seams.CreateProfile,
		BuildProvenance: func() BuildProvenance {
			if build == nil {
				return BuildProvenance{Version: "unknown", GitCommit: "unknown"}
			}
			return build()
		},
	}
}

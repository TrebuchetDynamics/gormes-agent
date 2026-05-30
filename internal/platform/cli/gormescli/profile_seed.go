package gormescli

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/profileseedruntime"
)

type ProfileSeedSeams = profileseedruntime.ProfileSeedSeams

func NewProfileSeedCommand(seams ProfileSeedSeams) *cobra.Command {
	return profileseedruntime.NewProfileSeedCommand(seams)
}

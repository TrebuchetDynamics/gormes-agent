package profile

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/profile/distribution"

const ProfileDistributionManifestFile = distribution.ProfileDistributionManifestFile

var ErrProfileDistributionInvalid = distribution.ErrProfileDistributionInvalid

type ProfileDistributionEnvRequirement = distribution.ProfileDistributionEnvRequirement
type ProfileDistributionManifest = distribution.ProfileDistributionManifest

func ReadProfileDistributionManifest(profileRoot string) (ProfileDistributionManifest, bool, error) {
	return distribution.ReadProfileDistributionManifest(profileRoot)
}

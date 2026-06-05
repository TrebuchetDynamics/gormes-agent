package update

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/update/releaseplan"

type UpdateInstallKind = releaseplan.UpdateInstallKind

const (
	UpdateInstallKindRelease         = releaseplan.UpdateInstallKindRelease
	UpdateInstallKindManagedSource   = releaseplan.UpdateInstallKindManagedSource
	UpdateInstallKindUnmanagedSource = releaseplan.UpdateInstallKindUnmanagedSource
	UpdateInstallKindUnknown         = releaseplan.UpdateInstallKindUnknown
)

type UpdateReleaseChannel = releaseplan.UpdateReleaseChannel

const (
	UpdateReleaseChannelStable      = releaseplan.UpdateReleaseChannelStable
	UpdateReleaseChannelDevelopment = releaseplan.UpdateReleaseChannelDevelopment
)

type UpdateSource = releaseplan.UpdateSource

const (
	UpdateSourceGitHubRelease = releaseplan.UpdateSourceGitHubRelease
	UpdateSourceManagedSource = releaseplan.UpdateSourceManagedSource
	UpdateSourceUnknown       = releaseplan.UpdateSourceUnknown
)

type UpdateBuildIdentity = releaseplan.UpdateBuildIdentity
type UpdateReleaseMetadata = releaseplan.UpdateReleaseMetadata

type UpdateReleaseComponent = releaseplan.UpdateReleaseComponent

const (
	UpdateReleaseComponentSnapshot       = releaseplan.UpdateReleaseComponentSnapshot
	UpdateReleaseComponentBinary         = releaseplan.UpdateReleaseComponentBinary
	UpdateReleaseComponentChecksum       = releaseplan.UpdateReleaseComponentChecksum
	UpdateReleaseComponentManifest       = releaseplan.UpdateReleaseComponentManifest
	UpdateReleaseComponentAssets         = releaseplan.UpdateReleaseComponentAssets
	UpdateReleaseComponentSkills         = releaseplan.UpdateReleaseComponentSkills
	UpdateReleaseComponentSourceCheckout = releaseplan.UpdateReleaseComponentSourceCheckout
)

type UpdateReleaseBlockerKind = releaseplan.UpdateReleaseBlockerKind

const (
	UpdateReleaseBlockerUnknownInstallLayout    = releaseplan.UpdateReleaseBlockerUnknownInstallLayout
	UpdateReleaseBlockerUnsupportedPlatform     = releaseplan.UpdateReleaseBlockerUnsupportedPlatform
	UpdateReleaseBlockerMissingReleaseMetadata  = releaseplan.UpdateReleaseBlockerMissingReleaseMetadata
	UpdateReleaseBlockerChannelMismatch         = releaseplan.UpdateReleaseBlockerChannelMismatch
	UpdateReleaseBlockerDirtyUnmanagedSource    = releaseplan.UpdateReleaseBlockerDirtyUnmanagedSource
	UpdateReleaseBlockerUnsupportedInstallState = releaseplan.UpdateReleaseBlockerUnsupportedInstallState
)

type UpdateReleaseBlocker = releaseplan.UpdateReleaseBlocker
type UpdateReleasePlanOptions = releaseplan.UpdateReleasePlanOptions
type UpdateReleasePlan = releaseplan.UpdateReleasePlan

func BuildUpdateReleasePlan(opts UpdateReleasePlanOptions) UpdateReleasePlan {
	return releaseplan.BuildUpdateReleasePlan(opts)
}

func ReleaseArtifactPlatformSlug(goos, goarch string) string {
	return releaseplan.ReleaseArtifactPlatformSlug(goos, goarch)
}

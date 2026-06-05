package cli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/profile"

var ErrActiveProfileUnset = profile.ErrActiveProfileUnset

func ReadActiveProfile(rootFile string) (string, error) { return profile.ReadActiveProfile(rootFile) }
func WriteActiveProfile(rootFile, name string) error {
	return profile.WriteActiveProfile(rootFile, name)
}
func ClearActiveProfile(rootFile string) error { return profile.ClearActiveProfile(rootFile) }

type ProfileContextScaffoldOptions = profile.ProfileContextScaffoldOptions
type ProfileContextScaffoldResult = profile.ProfileContextScaffoldResult

func MaterializeMainProfileContextScaffold(opts ProfileContextScaffoldOptions) (ProfileContextScaffoldResult, error) {
	return profile.MaterializeMainProfileContextScaffold(opts)
}
func ApplyProfileContextScaffold(opts ProfileContextScaffoldOptions) (ProfileContextScaffoldResult, error) {
	return profile.ApplyProfileContextScaffold(opts)
}

var ErrProfileCreateDefaultReserved = profile.ErrProfileCreateDefaultReserved
var ErrProfileCreateTargetExists = profile.ErrProfileCreateTargetExists
var ErrProfileCreateSourceMissing = profile.ErrProfileCreateSourceMissing

type ProfileCreateOptions = profile.ProfileCreateOptions
type ProfileCreateResult = profile.ProfileCreateResult

func CreateProfile(options ProfileCreateOptions) (ProfileCreateResult, error) {
	return profile.CreateProfile(options)
}

const ProfileDistributionManifestFile = profile.ProfileDistributionManifestFile

var ErrProfileDistributionInvalid = profile.ErrProfileDistributionInvalid

type ProfileDistributionEnvRequirement = profile.ProfileDistributionEnvRequirement
type ProfileDistributionManifest = profile.ProfileDistributionManifest

func ReadProfileDistributionManifest(profileRoot string) (ProfileDistributionManifest, bool, error) {
	return profile.ReadProfileDistributionManifest(profileRoot)
}

var ErrProfileNameEmpty = profile.ErrProfileNameEmpty
var ErrProfileNameTooLong = profile.ErrProfileNameTooLong
var ErrProfileNameInvalidChars = profile.ErrProfileNameInvalidChars
var ErrProfileNameReserved = profile.ErrProfileNameReserved

func ValidateProfileName(name string) error { return profile.ValidateProfileName(name) }

var ErrProfileXDGRootRequired = profile.ErrProfileXDGRootRequired

func ResolveProfileRoot(name string, gormesXDGConfigHome string) (string, error) {
	return profile.ResolveProfileRoot(name, gormesXDGConfigHome)
}

var ErrProfileBaseHomeRequired = profile.ErrProfileBaseHomeRequired

type ProfileStorageContract = profile.ProfileStorageContract

func NewProfileStorageContract(baseHome string) (ProfileStorageContract, error) {
	return profile.NewProfileStorageContract(baseHome)
}
func ResolveProfileRuntimeRoot(baseHome, name string) (string, error) {
	return profile.ResolveProfileRuntimeRoot(baseHome, name)
}

var ErrProfileRuntimeScopeMissing = profile.ErrProfileRuntimeScopeMissing

type ProfileRuntimeScope = profile.ProfileRuntimeScope
type ProfileRuntimeScopeOptions = profile.ProfileRuntimeScopeOptions

func ResolveProfileRuntimeScope(opts ProfileRuntimeScopeOptions) (ProfileRuntimeScope, error) {
	return profile.ResolveProfileRuntimeScope(opts)
}

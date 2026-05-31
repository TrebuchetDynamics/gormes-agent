package profile

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/profile/storage"

// ErrProfileBaseHomeRequired is returned when the homogeneous profile storage
// contract is constructed without the global Gormes base home (normally
// ~/.gormes). The contract is pure path math; callers remain responsible for
// resolving environment defaults and creating or migrating directories.
var ErrProfileBaseHomeRequired = storage.ErrProfileBaseHomeRequired

// ProfileStorageContract describes the target multi-profile storage layout for
// Gormes: global metadata lives under BaseHome and every runnable profile,
// including "main", lives under BaseHome/profiles/<name>.
type ProfileStorageContract = storage.ProfileStorageContract

// NewProfileStorageContract returns a pure path resolver for the homogeneous
// profile layout rooted at baseHome (for example ~/.gormes).
func NewProfileStorageContract(baseHome string) (ProfileStorageContract, error) {
	return storage.NewProfileStorageContract(baseHome)
}

// ResolveProfileRuntimeRoot returns the runnable profile root for baseHome and
// name without creating or migrating directories.
func ResolveProfileRuntimeRoot(baseHome, name string) (string, error) {
	return storage.ResolveProfileRuntimeRoot(baseHome, name)
}

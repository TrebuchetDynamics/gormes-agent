package config

import "github.com/TrebuchetDynamics/gormes-agent/internal/config/profilestorage"

// ProfileStorageScope identifies whether the active runtime storage root is the
// legacy base home or a profile-local runnable root.
type ProfileStorageScope = profilestorage.Scope

const (
	ProfileStorageScopeBaseHome    = profilestorage.ScopeBaseHome
	ProfileStorageScopeProfileRoot = profilestorage.ScopeProfileRoot
)

// ProfileStorageContract is the resolved path contract for profile-owned
// runtime data. It is pure path resolution: callers remain responsible for
// directory creation, migration, locking, and persistence.
type ProfileStorageContract = profilestorage.Contract

// CurrentProfileStorageContract resolves the storage contract for the current
// process GORMES_HOME while preserving the legacy profiles/main materialization
// probe used by MemoryDBPath and SessionDBPath.
func CurrentProfileStorageContract() ProfileStorageContract {
	return profilestorage.Current()
}

// NewProfileStorageContract resolves storage paths for gormesHome. If
// gormesHome is already a profile root, the scope is that profile. If it is a
// base home with a materialized profiles/main directory, main is the scope. All
// other base homes keep legacy root database paths.
func NewProfileStorageContract(gormesHome string) ProfileStorageContract {
	return profilestorage.New(gormesHome)
}

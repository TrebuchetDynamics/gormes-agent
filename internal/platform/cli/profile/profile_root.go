package profile

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/profile/storage"

// ErrProfileXDGRootRequired is returned when ResolveProfileRoot is called
// without a non-empty XDG config home; the helper refuses to invent a default
// so callers stay in charge of env resolution.
var ErrProfileXDGRootRequired = storage.ErrProfileXDGRootRequired

// ResolveProfileRoot maps a profile name and a caller-supplied XDG config home
// to the physical directory that holds that profile's Gormes state.
func ResolveProfileRoot(name string, gormesXDGConfigHome string) (string, error) {
	return storage.ResolveProfileRoot(name, gormesXDGConfigHome)
}

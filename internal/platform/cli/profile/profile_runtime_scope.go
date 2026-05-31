package profile

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/profile/scope"

// ErrProfileRuntimeScopeMissing is returned when the requested runtime profile
// is valid syntactically but absent from the known profile inventory supplied to
// the resolver.
var ErrProfileRuntimeScopeMissing = scope.ErrProfileRuntimeScopeMissing

// ErrSelectorHelperUnavailable is returned when profile runtime resolution is
// invoked without a required injected helper seam.
var ErrSelectorHelperUnavailable = scope.ErrSelectorHelperUnavailable

// ProfileRuntimeScope is the resolved per-profile boundary for a Gormes
// operation. It contains only identity and path data; applying environment
// variables or creating directories is the caller's responsibility.
type ProfileRuntimeScope = scope.ProfileRuntimeScope

// ReadActiveProfileNameFunc is the active-profile reader seam used by runtime
// scope resolution.
type ReadActiveProfileNameFunc = scope.ReadActiveProfileNameFunc

// ProfileRuntimeScopeOptions carries the external state needed to resolve a
// profile runtime scope without reading global process environment or touching
// the filesystem.
type ProfileRuntimeScopeOptions = scope.ProfileRuntimeScopeOptions

// ResolveProfileRuntimeScope derives the active profile identity and all
// profile-owned paths from explicit input, sticky active profile state, and the
// homogeneous ProfileStorageContract.
func ResolveProfileRuntimeScope(opts ProfileRuntimeScopeOptions) (ProfileRuntimeScope, error) {
	return scope.ResolveProfileRuntimeScope(opts)
}

package profile

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/profile/contextscaffold"

// ProfileContextScaffoldOptions names the profile root that should receive the
// editable context files Gormes exposes to operators.
type ProfileContextScaffoldOptions = contextscaffold.ProfileContextScaffoldOptions

// ProfileContextScaffoldResult reports which profile root was considered and
// the per-file template actions. Callers use this as operator evidence for
// setup, migration, and dry-run flows.
type ProfileContextScaffoldResult = contextscaffold.ProfileContextScaffoldResult

// MaterializeMainProfileContextScaffold materializes the built-in main profile
// root and seeds its editable context files.
func MaterializeMainProfileContextScaffold(opts ProfileContextScaffoldOptions) (ProfileContextScaffoldResult, error) {
	return contextscaffold.MaterializeMainProfileContextScaffold(opts)
}

// ApplyProfileContextScaffold seeds the default Gormes context templates into a
// profile root.
func ApplyProfileContextScaffold(opts ProfileContextScaffoldOptions) (ProfileContextScaffoldResult, error) {
	return contextscaffold.ApplyProfileContextScaffold(opts)
}

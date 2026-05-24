package profiles

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/cli/gormescli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/progress"
)

// SetupSections returns the Gormes-owned setup section metadata contributed by
// the profiles module. cmd/gormes still owns the dispatcher/chrome until the
// setup registry extraction is fully complete.
func SetupSections() []gormescli.SetupSection {
	return []gormescli.SetupSection{{
		Name:   "profiles",
		Label:  "Profiles",
		Module: progress.ModuleProfiles,
	}}
}

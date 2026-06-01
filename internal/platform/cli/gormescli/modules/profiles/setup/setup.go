package setup

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

// SetupSections returns the Gormes-owned setup section metadata contributed by
// the profiles module. cmd/gormes still owns the dispatcher/chrome until the
// setup registry extraction is fully complete.
func Sections() []gormescli.SetupSection {
	return []gormescli.SetupSection{{
		Name:   "profiles",
		Label:  "Profiles",
		Module: progress.ModuleProfiles,
	}}
}

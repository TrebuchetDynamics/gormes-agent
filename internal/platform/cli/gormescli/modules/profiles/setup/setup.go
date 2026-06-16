package setup

import (
	appsetup "github.com/TrebuchetDynamics/gormes-agent/internal/app/setup"
	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress"
)

// SetupSections returns the Gormes-owned setup section metadata contributed by
// the profiles module. cmd/gormes still owns the dispatcher/chrome until the
// setup registry extraction is fully complete.
func Sections() []appsetup.SetupSection {
	return []appsetup.SetupSection{{
		Name:   "profiles",
		Label:  "Profiles",
		Module: progress.ModuleProfiles,
	}}
}

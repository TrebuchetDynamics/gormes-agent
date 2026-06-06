package profiles

import (
	appsetup "github.com/TrebuchetDynamics/gormes-agent/internal/app/setup"
	profilesetup "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/profiles/setup"
)

// SetupSections returns the Gormes-owned setup section metadata contributed by
// the profiles module. The setup package owns the section registration data;
// this facade keeps the profiles module as the public CLI boundary.
func SetupSections() []appsetup.SetupSection {
	return profilesetup.Sections()
}

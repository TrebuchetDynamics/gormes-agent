package profiles

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	profilesetup "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/profiles/setup"
)

// SetupSections returns the Gormes-owned setup section metadata contributed by
// the profiles module. The setup package owns the section registration data;
// this facade keeps the profiles module as the public CLI boundary.
func SetupSections() []gormescli.SetupSection {
	return profilesetup.Sections()
}

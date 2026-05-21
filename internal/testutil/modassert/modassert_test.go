package modassert

import "testing"

func TestRequirePublicModuleVersionAcceptsCurrentModule(t *testing.T) {
	RequirePublicModuleVersion(t, "github.com/TrebuchetDynamics/gormes-agent", "")
}

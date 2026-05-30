package assets

import "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/apiserver/assets/manifest"

// DashboardAssetsManifest describes the front-end scaffold that can be built by
// Vite and embedded by the Go dashboard shell. It is intentionally static for
// this slice: later rows can replace RequiredAssets with the actual build
// manifest after page behavior is implemented.
type DashboardAssetsManifest = manifest.DashboardAssetsManifest

// DashboardDistManifest returns the deterministic dashboard scaffold contract
// used by tests and by future server-embedding work.
func DashboardDistManifest() DashboardAssetsManifest {
	return manifest.DashboardDistManifest()
}

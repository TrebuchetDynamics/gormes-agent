package assets

// DashboardAssetsManifest describes the front-end scaffold that can be built by
// Vite and embedded by the Go dashboard shell. It is intentionally static for
// this slice: later rows can replace RequiredAssets with the actual build
// manifest after page behavior is implemented.
type DashboardAssetsManifest struct {
	EntryHTML      string
	AppEntry       string
	Routes         []string
	RequiredAssets []string
}

// DashboardDistManifest returns the deterministic dashboard scaffold contract
// used by tests and by future server-embedding work.
func DashboardDistManifest() DashboardAssetsManifest {
	return DashboardAssetsManifest{
		EntryHTML: "index.html",
		AppEntry:  "src/main.tsx",
		Routes: []string{
			"/",
			"/chat",
			"/config",
			"/env",
			"/sessions",
			"/logs",
			"/cron",
			"/skills",
			"/docs",
			"/analytics",
		},
		RequiredAssets: []string{"web/package.json", "web/vite.config.ts", "web/src/App.tsx", "web/src/main.tsx"},
	}
}

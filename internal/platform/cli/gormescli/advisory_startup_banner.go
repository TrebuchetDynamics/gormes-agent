package gormescli

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/security"
)

// AdvisoryStartupBanner returns the security advisory startup banner for CLI
// entry points, or an empty string when all hits are acked or recently shown.
// It mirrors Hermes' startup_banner helper (security_advisories.py) while
// keeping the seam injectable so callers control the package-version detector.
//
// Production callers pass security.NoInstalledPackages as the installed seam
// (Go binary has no Python venv) and time.Now() as now. Tests inject an active
// seam to exercise the banner without modifying the default catalog.
func AdvisoryStartupBanner(gormesHome string, installed security.PackageVersionFunc, now time.Time) string {
	hits := security.DetectCompromised(security.DefaultCatalog(), installed)
	store := security.NewAckStore(gormesHome)
	acked, _ := store.AckedIDs()
	cache := security.NewBannerCache(gormesHome)
	return security.StartupBanner(hits, acked, cache, now)
}

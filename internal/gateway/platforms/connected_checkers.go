package platforms

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/platforms/connectivity"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/platforms/inventory"
)

// PlatformConnectionConfig is the minimal redacted config shape needed to
// decide whether a platform should appear as configured in status/readouts.
type PlatformConnectionConfig = connectivity.PlatformConnectionConfig

func PlatformLooksConfigured(cfg PlatformConnectionConfig) (bool, bool) {
	return connectivity.PlatformLooksConfigured(cfg)
}

func PlatformConnectedCheckerIDs() []string {
	return connectivity.PlatformConnectedCheckerIDs()
}

func MissingPlatformConnectedCheckers(manifest []PlatformManifestEntry) []string {
	return connectivity.MissingPlatformConnectedCheckers(toInventoryManifest(manifest))
}

func toInventoryManifest(manifest []PlatformManifestEntry) []inventory.PlatformManifestEntry {
	return manifest
}

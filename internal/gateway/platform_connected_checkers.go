package gateway

import gatewayplatforms "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/platforms"

// PlatformConnectionConfig is the minimal redacted config shape needed to
// decide whether a platform should appear as configured in status/readouts.
type PlatformConnectionConfig = gatewayplatforms.PlatformConnectionConfig

func PlatformLooksConfigured(cfg PlatformConnectionConfig) (bool, bool) {
	return gatewayplatforms.PlatformLooksConfigured(cfg)
}

func PlatformConnectedCheckerIDs() []string {
	return gatewayplatforms.PlatformConnectedCheckerIDs()
}

func MissingPlatformConnectedCheckers(manifest []PlatformManifestEntry) []string {
	return gatewayplatforms.MissingPlatformConnectedCheckers(manifest)
}

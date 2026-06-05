package delivery

import "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/delivery/routing"

// Target is a parsed --deliver destination.
type Target = routing.Target

// HomeTargets maps a platform name to its configured channel-neutral home
// target. Values usually come from Hermes platforms.<name>.home_channel.
type HomeTargets = routing.HomeTargets

// OriginSource is the minimal platform/chat/thread source needed to resolve
// origin and discovery delivery targets without depending on gateway runtime
// types.
type OriginSource = routing.OriginSource

// HomeDiscoveryFallback carries a discovery-owned source that can be used as
// the home channel only when that platform explicitly allows discovery.
type HomeDiscoveryFallback = routing.HomeDiscoveryFallback

// MissingHomeError is returned when a platform-only delivery target has no
// explicit, configured, or discovery-approved home channel.
type MissingHomeError = routing.MissingHomeError

// ResolveHomeTarget expands a platform-only delivery target through the shared
// home-channel resolver while preserving the legacy bool result contract.
func ResolveHomeTarget(target Target, homes HomeTargets) (Target, bool) {
	return routing.ResolveHomeTarget(target, homes)
}

// ResolveHomeTargetWithFallback resolves platform-name targets through the
// shared home-channel hierarchy.
func ResolveHomeTargetWithFallback(target Target, homes HomeTargets, fallback HomeDiscoveryFallback) (Target, error) {
	return routing.ResolveHomeTargetWithFallback(target, homes, fallback)
}

// ParseTarget converts a single --deliver token into a typed target.
func ParseTarget(raw string, origin *OriginSource) (Target, error) {
	return routing.ParseTarget(raw, origin)
}

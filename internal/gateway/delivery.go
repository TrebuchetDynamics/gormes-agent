package gateway

import gatewaydelivery "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/delivery"

// DeliveryTarget is a parsed --deliver destination.
type DeliveryTarget = gatewaydelivery.Target

// HomeChannelTargets maps a platform name to its configured channel-neutral
// home target. Values usually come from Hermes platforms.<name>.home_channel.
type HomeChannelTargets = gatewaydelivery.HomeTargets

// HomeChannelDiscoveryFallback carries a discovery-owned source that can be
// used as the home channel only when that platform explicitly allows discovery.
type HomeChannelDiscoveryFallback struct {
	Source           SessionSource
	DiscoveryEnabled bool
}

// MissingHomeChannelError is returned when a platform-only delivery target has
// no explicit, configured, or discovery-approved home channel.
type MissingHomeChannelError = gatewaydelivery.MissingHomeError

// ResolveHomeChannelTarget expands a platform-only delivery target (for
// example "discord") to that platform's configured home channel. Explicit
// targets, origin, and local remain unchanged so callers can preserve per-turn
// source routing and user-specified destinations. The bool return is false
// when no configured home exists; callers that need degradation evidence should
// use ResolveHomeChannelTargetWithFallback.
func ResolveHomeChannelTarget(target DeliveryTarget, homes HomeChannelTargets) (DeliveryTarget, bool) {
	return gatewaydelivery.ResolveHomeTarget(target, homes)
}

// ResolveHomeChannelTargetWithFallback resolves platform-name targets through
// the shared home-channel hierarchy: explicit/origin/local targets are already
// resolved, configured home_channel wins, discovery-owned source follows only
// when enabled, and missing homes return MissingHomeChannelError.
func ResolveHomeChannelTargetWithFallback(target DeliveryTarget, homes HomeChannelTargets, fallback HomeChannelDiscoveryFallback) (DeliveryTarget, error) {
	return gatewaydelivery.ResolveHomeTargetWithFallback(target, homes, gatewaydelivery.HomeDiscoveryFallback{
		Source: gatewaydelivery.OriginSource{
			Platform: fallback.Source.Platform,
			ChatID:   fallback.Source.ChatID,
			ThreadID: fallback.Source.ThreadID,
		},
		DiscoveryEnabled: fallback.DiscoveryEnabled,
	})
}

// ParseDeliveryTarget converts a single --deliver token into a typed target.
// Parsing is syntax-only; runtime availability checks happen later when a
// router binds targets to concrete platform channels.
func ParseDeliveryTarget(raw string, origin *SessionSource) (DeliveryTarget, error) {
	var deliveryOrigin *gatewaydelivery.OriginSource
	if origin != nil {
		deliveryOrigin = &gatewaydelivery.OriginSource{
			Platform: origin.Platform,
			ChatID:   origin.ChatID,
			ThreadID: origin.ThreadID,
		}
	}
	return gatewaydelivery.ParseTarget(raw, deliveryOrigin)
}

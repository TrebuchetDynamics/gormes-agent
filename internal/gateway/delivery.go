package gateway

import (
	"errors"
	"fmt"
	"strings"
)

// DeliveryTarget is a parsed --deliver destination.
type DeliveryTarget struct {
	Platform   string
	ChatID     string
	ThreadID   string
	IsOrigin   bool
	IsExplicit bool
}

// HomeChannelTargets maps a platform name to its configured channel-neutral
// home target. Values usually come from Hermes platforms.<name>.home_channel.
type HomeChannelTargets map[string]DeliveryTarget

// HomeChannelDiscoveryFallback carries a discovery-owned source that can be
// used as the home channel only when that platform explicitly allows discovery.
type HomeChannelDiscoveryFallback struct {
	Source           SessionSource
	DiscoveryEnabled bool
}

// MissingHomeChannelError is returned when a platform-only delivery target has
// no explicit, configured, or discovery-approved home channel.
type MissingHomeChannelError struct {
	Platform string
}

func (e MissingHomeChannelError) Error() string {
	if e.Platform == "" {
		return "gateway: missing home channel"
	}
	return fmt.Sprintf("gateway: missing home channel for %s", e.Platform)
}

func (t DeliveryTarget) String() string {
	if t.IsOrigin {
		return "origin"
	}
	platform := strings.ToLower(strings.TrimSpace(t.Platform))
	if platform == "local" || platform == "" {
		return "local"
	}
	if t.ChatID == "" {
		return platform
	}
	if t.ThreadID == "" {
		return platform + ":" + t.ChatID
	}
	return platform + ":" + t.ChatID + ":" + t.ThreadID
}

// ResolveHomeChannelTarget expands a platform-only delivery target (for
// example "discord") to that platform's configured home channel. Explicit
// targets, origin, and local remain unchanged so callers can preserve per-turn
// source routing and user-specified destinations. The bool return is false
// when no configured home exists; callers that need degradation evidence should
// use ResolveHomeChannelTargetWithFallback.
func ResolveHomeChannelTarget(target DeliveryTarget, homes HomeChannelTargets) (DeliveryTarget, bool) {
	resolved, err := ResolveHomeChannelTargetWithFallback(target, homes, HomeChannelDiscoveryFallback{})
	if err != nil {
		return target, false
	}
	return resolved, resolved != target
}

// ResolveHomeChannelTargetWithFallback resolves platform-name targets through
// the shared home-channel hierarchy: explicit/origin/local targets are already
// resolved, configured home_channel wins, discovery-owned source follows only
// when enabled, and missing homes return MissingHomeChannelError.
func ResolveHomeChannelTargetWithFallback(target DeliveryTarget, homes HomeChannelTargets, fallback HomeChannelDiscoveryFallback) (DeliveryTarget, error) {
	if target.IsOrigin || target.IsExplicit || strings.EqualFold(target.Platform, "local") || strings.TrimSpace(target.ChatID) != "" {
		return target, nil
	}
	platform := strings.ToLower(strings.TrimSpace(target.Platform))
	if platform == "" {
		return target, MissingHomeChannelError{}
	}
	if homes != nil {
		if home, ok := homes[platform]; ok && strings.TrimSpace(home.ChatID) != "" {
			home.Platform = strings.ToLower(strings.TrimSpace(firstNonEmptyString(home.Platform, platform)))
			home.ChatID = strings.TrimSpace(home.ChatID)
			home.ThreadID = strings.TrimSpace(home.ThreadID)
			return home, nil
		}
	}
	if fallback.DiscoveryEnabled && strings.EqualFold(fallback.Source.Platform, platform) && strings.TrimSpace(fallback.Source.ChatID) != "" {
		return DeliveryTarget{
			Platform: platform,
			ChatID:   strings.TrimSpace(fallback.Source.ChatID),
			ThreadID: strings.TrimSpace(fallback.Source.ThreadID),
		}, nil
	}
	return target, MissingHomeChannelError{Platform: platform}
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// ParseDeliveryTarget converts a single --deliver token into a typed target.
// Parsing is syntax-only; runtime availability checks happen later when a
// router binds targets to concrete platform channels.
func ParseDeliveryTarget(raw string, origin *SessionSource) (DeliveryTarget, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return DeliveryTarget{}, errors.New("gateway: empty delivery target")
	}
	if strings.EqualFold(trimmed, "origin") {
		if origin == nil {
			return DeliveryTarget{Platform: "local", IsOrigin: true}, nil
		}
		return DeliveryTarget{
			Platform: strings.ToLower(strings.TrimSpace(origin.Platform)),
			ChatID:   strings.TrimSpace(origin.ChatID),
			ThreadID: strings.TrimSpace(origin.ThreadID),
			IsOrigin: true,
		}, nil
	}
	if strings.EqualFold(trimmed, "local") {
		return DeliveryTarget{Platform: "local"}, nil
	}

	parts := strings.Split(trimmed, ":")
	switch len(parts) {
	case 1:
		platform := strings.ToLower(strings.TrimSpace(parts[0]))
		if platform == "" {
			return DeliveryTarget{}, errors.New("gateway: empty delivery platform")
		}
		return DeliveryTarget{Platform: platform}, nil
	case 2:
		platform := strings.ToLower(strings.TrimSpace(parts[0]))
		chatID := strings.TrimSpace(parts[1])
		if platform == "" || chatID == "" {
			return DeliveryTarget{}, errors.New("gateway: invalid explicit delivery target")
		}
		return DeliveryTarget{Platform: platform, ChatID: chatID, IsExplicit: true}, nil
	case 3:
		platform := strings.ToLower(strings.TrimSpace(parts[0]))
		chatID := strings.TrimSpace(parts[1])
		threadID := strings.TrimSpace(parts[2])
		if platform == "" || chatID == "" || threadID == "" {
			return DeliveryTarget{}, errors.New("gateway: invalid threaded delivery target")
		}
		return DeliveryTarget{
			Platform:   platform,
			ChatID:     chatID,
			ThreadID:   threadID,
			IsExplicit: true,
		}, nil
	default:
		return DeliveryTarget{}, errors.New("gateway: invalid delivery target")
	}
}

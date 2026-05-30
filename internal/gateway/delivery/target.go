package delivery

import (
	"errors"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/textvalue"
)

// Target is a parsed --deliver destination.
type Target struct {
	Platform   string
	ChatID     string
	ThreadID   string
	IsOrigin   bool
	IsExplicit bool
}

// HomeTargets maps a platform name to its configured channel-neutral home
// target. Values usually come from Hermes platforms.<name>.home_channel.
type HomeTargets map[string]Target

// OriginSource is the minimal platform/chat/thread source needed to resolve
// origin and discovery delivery targets without depending on gateway runtime
// types.
type OriginSource struct {
	Platform string
	ChatID   string
	ThreadID string
}

// HomeDiscoveryFallback carries a discovery-owned source that can be used as
// the home channel only when that platform explicitly allows discovery.
type HomeDiscoveryFallback struct {
	Source           OriginSource
	DiscoveryEnabled bool
}

// MissingHomeError is returned when a platform-only delivery target has no
// explicit, configured, or discovery-approved home channel.
type MissingHomeError struct {
	Platform string
}

func (e MissingHomeError) Error() string {
	if e.Platform == "" {
		return "gateway: missing home channel"
	}
	return fmt.Sprintf("gateway: missing home channel for %s", e.Platform)
}

func (t Target) String() string {
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

// ResolveHomeTarget expands a platform-only delivery target (for example
// "discord") to that platform's configured home channel. Explicit targets,
// origin, and local remain unchanged so callers can preserve per-turn source
// routing and user-specified destinations. The bool return is false when no
// configured home exists; callers that need degradation evidence should use
// ResolveHomeTargetWithFallback.
func ResolveHomeTarget(target Target, homes HomeTargets) (Target, bool) {
	resolved, err := ResolveHomeTargetWithFallback(target, homes, HomeDiscoveryFallback{})
	if err != nil {
		return target, false
	}
	return resolved, resolved != target
}

// ResolveHomeTargetWithFallback resolves platform-name targets through the
// shared home-channel hierarchy: explicit/origin/local targets are already
// resolved, configured home_channel wins, discovery-owned source follows only
// when enabled, and missing homes return MissingHomeError.
func ResolveHomeTargetWithFallback(target Target, homes HomeTargets, fallback HomeDiscoveryFallback) (Target, error) {
	if target.IsOrigin || target.IsExplicit || strings.EqualFold(target.Platform, "local") || strings.TrimSpace(target.ChatID) != "" {
		return target, nil
	}
	platform := strings.ToLower(strings.TrimSpace(target.Platform))
	if platform == "" {
		return target, MissingHomeError{}
	}
	if homes != nil {
		if home, ok := homes[platform]; ok && strings.TrimSpace(home.ChatID) != "" {
			home.Platform = strings.ToLower(textvalue.FirstNonEmptyTrimmed(home.Platform, platform))
			home.ChatID = strings.TrimSpace(home.ChatID)
			home.ThreadID = strings.TrimSpace(home.ThreadID)
			return home, nil
		}
	}
	if fallback.DiscoveryEnabled && strings.EqualFold(fallback.Source.Platform, platform) && strings.TrimSpace(fallback.Source.ChatID) != "" {
		return Target{
			Platform: platform,
			ChatID:   strings.TrimSpace(fallback.Source.ChatID),
			ThreadID: strings.TrimSpace(fallback.Source.ThreadID),
		}, nil
	}
	return target, MissingHomeError{Platform: platform}
}

// ParseTarget converts a single --deliver token into a typed target. Parsing is
// syntax-only; runtime availability checks happen later when a router binds
// targets to concrete platform channels.
func ParseTarget(raw string, origin *OriginSource) (Target, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return Target{}, errors.New("gateway: empty delivery target")
	}
	if strings.EqualFold(trimmed, "origin") {
		if origin == nil {
			return Target{Platform: "local", IsOrigin: true}, nil
		}
		return Target{
			Platform: strings.ToLower(strings.TrimSpace(origin.Platform)),
			ChatID:   strings.TrimSpace(origin.ChatID),
			ThreadID: strings.TrimSpace(origin.ThreadID),
			IsOrigin: true,
		}, nil
	}
	if strings.EqualFold(trimmed, "local") {
		return Target{Platform: "local"}, nil
	}

	parts := strings.Split(trimmed, ":")
	switch len(parts) {
	case 1:
		platform := strings.ToLower(strings.TrimSpace(parts[0]))
		if platform == "" {
			return Target{}, errors.New("gateway: empty delivery platform")
		}
		return Target{Platform: platform}, nil
	case 2:
		platform := strings.ToLower(strings.TrimSpace(parts[0]))
		chatID := strings.TrimSpace(parts[1])
		if platform == "" || chatID == "" {
			return Target{}, errors.New("gateway: invalid explicit delivery target")
		}
		return Target{Platform: platform, ChatID: chatID, IsExplicit: true}, nil
	case 3:
		platform := strings.ToLower(strings.TrimSpace(parts[0]))
		chatID := strings.TrimSpace(parts[1])
		threadID := strings.TrimSpace(parts[2])
		if platform == "" || chatID == "" || threadID == "" {
			return Target{}, errors.New("gateway: invalid threaded delivery target")
		}
		return Target{Platform: platform, ChatID: chatID, ThreadID: threadID, IsExplicit: true}, nil
	default:
		return Target{}, errors.New("gateway: invalid delivery target")
	}
}

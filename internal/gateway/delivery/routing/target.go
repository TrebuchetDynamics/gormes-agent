package routing

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/delivery/address"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/textvalue"
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
	platform := address.Platform(t.Platform)
	if platform == "local" || platform == "" {
		return "local"
	}
	if t.ChatID == "" {
		return platform
	}
	if platform != "matrix" && platform != "simplex" && (strings.Contains(t.ChatID, ":") || strings.Contains(t.ThreadID, ":")) {
		out := platform + ":" + strconv.Itoa(len(t.ChatID)) + ":" + t.ChatID
		if t.ThreadID != "" {
			out += ":" + strconv.Itoa(len(t.ThreadID)) + ":" + t.ThreadID
		}
		return out
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
	if target.IsOrigin || target.IsExplicit || strings.EqualFold(target.Platform, "local") || address.ID(target.ChatID) != "" {
		return target, nil
	}
	platform := address.Platform(target.Platform)
	if platform == "" {
		return target, MissingHomeError{}
	}
	if home, ok := configuredHomeTarget(platform, homes); ok {
		return home, nil
	}
	if fallback.DiscoveryEnabled && address.Platform(fallback.Source.Platform) == platform && address.ID(fallback.Source.ChatID) != "" {
		return Target{
			Platform:   platform,
			ChatID:     address.ID(fallback.Source.ChatID),
			ThreadID:   address.ID(fallback.Source.ThreadID),
			IsExplicit: true,
		}, nil
	}
	return target, MissingHomeError{Platform: platform}
}

func configuredHomeTarget(platform string, homes HomeTargets) (Target, bool) {
	if homes == nil {
		return Target{}, false
	}
	if home, ok := normalizeConfiguredHomeTarget(platform, homes[platform]); ok {
		return home, true
	}
	keys := make([]string, 0, len(homes))
	for key := range homes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if address.Platform(key) != platform {
			continue
		}
		if home, ok := normalizeConfiguredHomeTarget(platform, homes[key]); ok {
			return home, true
		}
	}
	return Target{}, false
}

func normalizeConfiguredHomeTarget(platform string, home Target) (Target, bool) {
	if address.ID(home.ChatID) == "" {
		return Target{}, false
	}
	home.Platform = address.Platform(textvalue.FirstNonEmptyTrimmed(home.Platform, platform))
	home.ChatID = address.ID(home.ChatID)
	home.ThreadID = address.ID(home.ThreadID)
	home.IsExplicit = true
	return home, true
}

// ParseTarget converts a single --deliver token into a typed target. Parsing is
// syntax-only; runtime availability checks happen later when a router binds
// targets to concrete platform channels.
func ParseTarget(raw string, origin *OriginSource) (Target, error) {
	trimmed := address.ID(raw)
	if trimmed == "" {
		return Target{}, errors.New("gateway: empty delivery target")
	}
	if strings.EqualFold(trimmed, "origin") {
		if origin == nil {
			return Target{Platform: "local", IsOrigin: true}, nil
		}
		return Target{
			Platform: address.Platform(origin.Platform),
			ChatID:   address.ID(origin.ChatID),
			ThreadID: address.ID(origin.ThreadID),
			IsOrigin: true,
		}, nil
	}
	if strings.EqualFold(trimmed, "local") {
		return Target{Platform: "local"}, nil
	}

	parts := strings.Split(trimmed, ":")
	if target, ok, err := parseLengthEncodedTarget(parts); ok || err != nil {
		return target, err
	}
	if len(parts) > 0 && address.Platform(parts[0]) == "matrix" && len(parts) >= 4 {
		return parseMatrixColonTarget(parts)
	}
	if len(parts) > 0 && address.Platform(parts[0]) == "simplex" && len(parts) >= 3 && address.ID(parts[1]) == "group" {
		return parseSimplexGroupTarget(parts)
	}
	switch len(parts) {
	case 1:
		platform := address.Platform(parts[0])
		if platform == "" {
			return Target{}, errors.New("gateway: empty delivery platform")
		}
		return Target{Platform: platform}, nil
	case 2:
		platform := address.Platform(parts[0])
		chatID := address.ID(parts[1])
		if platform == "" || chatID == "" {
			return Target{}, errors.New("gateway: invalid explicit delivery target")
		}
		return Target{Platform: platform, ChatID: chatID, IsExplicit: true}, nil
	case 3:
		platform := address.Platform(parts[0])
		if platformColonChatIDPrefixAllowed(platform, address.ID(parts[1])) {
			chatID := address.ID(strings.Join(parts[1:], ":"))
			if chatID == "" {
				return Target{}, errors.New("gateway: invalid explicit delivery target")
			}
			return Target{Platform: platform, ChatID: chatID, IsExplicit: true}, nil
		}
		chatID := address.ID(parts[1])
		threadID := address.ID(parts[2])
		if platform == "" || chatID == "" || threadID == "" {
			return Target{}, errors.New("gateway: invalid threaded delivery target")
		}
		return Target{Platform: platform, ChatID: chatID, ThreadID: threadID, IsExplicit: true}, nil
	default:
		return Target{}, errors.New("gateway: invalid delivery target")
	}
}

func parseLengthEncodedTarget(parts []string) (Target, bool, error) {
	if len(parts) < 4 {
		return Target{}, false, nil
	}
	platform := address.Platform(parts[0])
	if platform == "" {
		return Target{}, false, nil
	}
	chatLen, err := strconv.Atoi(address.ID(parts[1]))
	if err != nil || chatLen <= 0 {
		return Target{}, false, nil
	}
	rest := strings.Join(parts[2:], ":")
	if len(rest) < chatLen {
		return Target{}, true, errors.New("gateway: invalid length-encoded delivery target")
	}
	chatID := rest[:chatLen]
	remainder := rest[chatLen:]
	if !utf8.ValidString(chatID) || address.ID(chatID) == "" {
		return Target{}, true, errors.New("gateway: invalid length-encoded delivery target")
	}
	if remainder == "" {
		return Target{Platform: platform, ChatID: chatID, IsExplicit: true}, true, nil
	}
	if !strings.HasPrefix(remainder, ":") {
		return Target{}, true, errors.New("gateway: invalid length-encoded delivery target")
	}
	remainder = strings.TrimPrefix(remainder, ":")
	threadLenText, threadValue, ok := strings.Cut(remainder, ":")
	if !ok {
		return Target{}, true, errors.New("gateway: invalid length-encoded delivery target")
	}
	threadLen, err := strconv.Atoi(address.ID(threadLenText))
	if err != nil || threadLen <= 0 || len(threadValue) != threadLen || !utf8.ValidString(threadValue) || address.ID(threadValue) == "" {
		return Target{}, true, errors.New("gateway: invalid length-encoded delivery target")
	}
	return Target{Platform: platform, ChatID: chatID, ThreadID: threadValue, IsExplicit: true}, true, nil
}

func parseSimplexGroupTarget(parts []string) (Target, error) {
	chatID := address.ID(strings.Join(parts[1:], ":"))
	if chatID == "" || !strings.HasPrefix(chatID, "group:") || strings.TrimPrefix(chatID, "group:") == "" {
		return Target{}, errors.New("gateway: invalid explicit delivery target")
	}
	return Target{Platform: "simplex", ChatID: chatID, IsExplicit: true}, nil
}

func parseMatrixColonTarget(parts []string) (Target, error) {
	for _, part := range parts[1:] {
		if address.ID(part) == "" {
			return Target{}, errors.New("gateway: invalid explicit delivery target")
		}
	}
	threadStart := 0
	for i := 2; i < len(parts); i++ {
		if strings.HasPrefix(address.ID(parts[i]), "$") {
			threadStart = i
			break
		}
	}
	if threadStart == 0 {
		chatID := address.ID(strings.Join(parts[1:], ":"))
		if chatID == "" || !strings.HasPrefix(chatID, "!") {
			return Target{}, errors.New("gateway: invalid explicit delivery target")
		}
		return Target{Platform: "matrix", ChatID: chatID, IsExplicit: true}, nil
	}
	chatID := address.ID(strings.Join(parts[1:threadStart], ":"))
	threadID := address.ID(strings.Join(parts[threadStart:], ":"))
	if chatID == "" || threadID == "" || !strings.HasPrefix(chatID, "!") {
		return Target{}, errors.New("gateway: invalid threaded delivery target")
	}
	return Target{Platform: "matrix", ChatID: chatID, ThreadID: threadID, IsExplicit: true}, nil
}

func platformColonChatIDPrefixAllowed(platform, firstChatPart string) bool {
	switch platform {
	case "matrix":
		return strings.HasPrefix(firstChatPart, "!")
	case "simplex":
		return firstChatPart == "group"
	default:
		return false
	}
}

package routing

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"
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
	if targetStringNeedsLengthEncoding(platform, t.ChatID, t.ThreadID) {
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

func targetStringNeedsLengthEncoding(platform, chatID, threadID string) bool {
	if platform == "simplex" && threadID != "" {
		return true
	}
	if platform == "matrix" {
		return threadID != "" && (strings.Contains(chatID, ":") || strings.Contains(threadID, ":") || !strings.HasPrefix(threadID, "$"))
	}
	return platform != "simplex" && (strings.Contains(chatID, ":") || strings.Contains(threadID, ":"))
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
		chatID := address.ID(fallback.Source.ChatID)
		threadID := address.ID(fallback.Source.ThreadID)
		if !containsControlRune(chatID) && !containsControlRune(threadID) {
			return Target{
				Platform:   platform,
				ChatID:     chatID,
				ThreadID:   threadID,
				IsExplicit: true,
			}, nil
		}
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
	home.Platform = address.Platform(textvalue.FirstNonEmptyTrimmed(home.Platform, platform))
	home.ChatID = address.ID(home.ChatID)
	home.ThreadID = address.ID(home.ThreadID)
	if home.Platform != platform || home.ChatID == "" || containsControlRune(home.Platform) || containsControlRune(home.ChatID) || containsControlRune(home.ThreadID) {
		return Target{}, false
	}
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
	if containsControlRune(trimmed) {
		return Target{}, errors.New("gateway: invalid delivery target")
	}
	if strings.EqualFold(trimmed, "origin") {
		if origin == nil {
			return Target{Platform: "local", IsOrigin: true}, nil
		}
		platform := address.Platform(origin.Platform)
		chatID := address.ID(origin.ChatID)
		threadID := address.ID(origin.ThreadID)
		if containsControlRune(platform) || containsControlRune(chatID) || containsControlRune(threadID) {
			return Target{}, errors.New("gateway: invalid origin delivery target")
		}
		if platform == "" || chatID == "" {
			return Target{Platform: "local", IsOrigin: true}, nil
		}
		return Target{
			Platform: platform,
			ChatID:   chatID,
			ThreadID: threadID,
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
	if len(parts) > 0 && address.Platform(parts[0]) == "matrix" && len(parts) >= 3 {
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
		if platform == "" || platform == "local" || chatID == "" {
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
		if platform == "" || platform == "local" || chatID == "" || threadID == "" {
			return Target{}, errors.New("gateway: invalid threaded delivery target")
		}
		return Target{Platform: platform, ChatID: chatID, ThreadID: threadID, IsExplicit: true}, nil
	default:
		return Target{}, errors.New("gateway: invalid delivery target")
	}
}

func containsControlRune(value string) bool {
	for _, r := range value {
		if unicode.IsControl(r) || hiddenFormattingRune(r) {
			return true
		}
	}
	return false
}

func hiddenFormattingRune(r rune) bool {
	switch {
	case r >= 0x200b && r <= 0x200f:
		return true
	case r >= 0x2028 && r <= 0x202e:
		return true
	case r >= 0x2060 && r <= 0x2069:
		return true
	case r == 0xfeff || r == 0xfffc:
		return true
	case r >= 0xfff9 && r <= 0xfffb:
		return true
	default:
		return false
	}
}

func parseLengthEncodedTarget(parts []string) (Target, bool, error) {
	if len(parts) < 4 {
		return Target{}, false, nil
	}
	platform := address.Platform(parts[0])
	if platform == "" || platform == "local" {
		return Target{}, false, nil
	}
	chatLen, lengthLooksEncoded, err := parsePositiveTargetLength(parts[1])
	if err != nil {
		return Target{}, true, errors.New("gateway: invalid length-encoded delivery target")
	}
	if !lengthLooksEncoded {
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
	chatID = address.ID(chatID)
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
	threadLen, lengthLooksEncoded, err := parsePositiveTargetLength(threadLenText)
	threadID := address.ID(threadValue)
	if err != nil || !lengthLooksEncoded || len(threadValue) != threadLen || !utf8.ValidString(threadValue) || threadID == "" {
		return Target{}, true, errors.New("gateway: invalid length-encoded delivery target")
	}
	return Target{Platform: platform, ChatID: chatID, ThreadID: threadID, IsExplicit: true}, true, nil
}

func parsePositiveTargetLength(raw string) (int, bool, error) {
	text := address.ID(raw)
	if text == "" {
		return 0, false, nil
	}
	for _, r := range text {
		if r < '0' || r > '9' {
			return 0, false, nil
		}
	}
	length, err := strconv.Atoi(text)
	if err != nil || length <= 0 {
		return 0, true, err
	}
	return length, true, nil
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

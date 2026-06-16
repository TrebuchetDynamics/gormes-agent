package readmodel

import (
	"strings"

	gatewaydelivery "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/delivery"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model"
)

// Resolve resolves a human-friendly channel target for platform into a concrete
// gatewaydelivery.Target. It follows Hermes matching order: exact ID, exact display/name,
// Discord guild-qualified name, then unambiguous prefix.
func (d Directory) Resolve(platform, query string) (gatewaydelivery.Target, model.Evidence) {
	platform = model.NormalizePlatform(platform)
	raw := strings.TrimSpace(query)
	entries := d.Platforms[platform]
	if platform == "" || raw == "" || len(entries) == 0 {
		return gatewaydelivery.Target{}, model.Evidence{Code: model.EvidenceChannelDirectoryMissing, Platform: platform, Query: raw}
	}
	for _, entry := range entries {
		if entry.ID == raw {
			return resolvedEntryTarget(platform, raw, entry)
		}
	}
	normalized := model.NormalizeQuery(raw)
	if target, evidence, ok := resolveDirectoryMatches(platform, raw, exactNameMatches(platform, entries, normalized)); ok {
		return target, evidence
	}
	if guildPart, channelPart, ok := guildQualifiedQueryParts(raw); ok {
		if target, evidence, ok := resolveDirectoryMatches(platform, raw, guildQualifiedMatches(entries, guildPart, channelPart)); ok {
			return target, evidence
		}
	}
	if target, evidence, ok := resolveDirectoryMatches(platform, raw, prefixNameMatches(entries, normalized)); ok {
		return target, evidence
	}
	return gatewaydelivery.Target{}, model.Evidence{Code: model.EvidenceChannelDirectoryMissing, Platform: platform, Query: raw}
}

func exactNameMatches(platform string, entries []model.Entry, normalized string) []model.Entry {
	return collectMatches(entries, func(entry model.Entry) bool {
		return model.NormalizeQuery(entry.Name) == normalized ||
			model.NormalizeQuery(model.TargetDisplayName(platform, entry)) == normalized ||
			model.NormalizeQuery(displayDirectoryText(model.TargetDisplayName(platform, entry))) == normalized
	})
}

func guildQualifiedQueryParts(raw string) (guildPart, channelPart string, ok bool) {
	parts := strings.Split(strings.TrimSpace(raw), "/")
	if len(parts) < 2 {
		return "", "", false
	}
	guildPart = model.NormalizeGuildQuery(strings.Join(parts[:len(parts)-1], "/"))
	channelPart = model.NormalizeQuery(parts[len(parts)-1])
	return guildPart, channelPart, guildPart != "" && channelPart != ""
}

func guildQualifiedMatches(entries []model.Entry, guildPart, channelPart string) []model.Entry {
	return collectMatches(entries, func(entry model.Entry) bool {
		guildMatches := model.NormalizeGuildQuery(model.EntryGuild(entry)) == guildPart ||
			model.NormalizeGuildQuery(displayDirectoryText(model.EntryGuild(entry))) == guildPart
		channelMatches := model.NormalizeQuery(entry.Name) == channelPart ||
			model.NormalizeQuery(displayDirectoryText(entry.Name)) == channelPart
		return guildMatches && channelMatches
	})
}

func prefixNameMatches(entries []model.Entry, normalized string) []model.Entry {
	return collectMatches(entries, func(entry model.Entry) bool {
		return strings.HasPrefix(model.NormalizeQuery(entry.Name), normalized) ||
			strings.HasPrefix(model.NormalizeQuery(displayDirectoryText(entry.Name)), normalized)
	})
}

func collectMatches(entries []model.Entry, keep func(model.Entry) bool) []model.Entry {
	matches := make([]model.Entry, 0, 1)
	for _, entry := range entries {
		if keep(entry) {
			matches = append(matches, entry)
		}
	}
	return matches
}

func resolveDirectoryMatches(platform, raw string, matches []model.Entry) (gatewaydelivery.Target, model.Evidence, bool) {
	validTargets := make([]gatewaydelivery.Target, 0, len(matches))
	for _, match := range matches {
		target, evidence := resolvedEntryTarget(platform, raw, match)
		if evidence.Code != "" {
			continue
		}
		validTargets = append(validTargets, target)
	}
	switch len(validTargets) {
	case 0:
		if len(matches) == 0 {
			return gatewaydelivery.Target{}, model.Evidence{}, false
		}
		return gatewaydelivery.Target{}, model.Evidence{Code: model.EvidenceChannelDirectoryMissing, Platform: platform, Query: raw}, true
	case 1:
		return validTargets[0], model.Evidence{}, true
	default:
		return gatewaydelivery.Target{}, model.Evidence{Code: model.EvidenceChannelTargetAmbiguous, Platform: platform, Query: raw}, true
	}
}

func resolvedEntryTarget(platform, raw string, entry model.Entry) (gatewaydelivery.Target, model.Evidence) {
	target := model.DeliveryTarget(platform, entry)
	if target.ChatID == "" || !target.IsExplicit {
		return gatewaydelivery.Target{}, model.Evidence{Code: model.EvidenceChannelDirectoryMissing, Platform: platform, Query: raw}
	}
	return target, model.Evidence{}
}

// ValidateDeliveryTarget returns channel_target_stale when an explicit target
// no longer appears in a refreshed platform directory. Platform-only, local,
// origin, and unknown-directory targets are left to existing home-channel and
// missing-directory resolution paths.
func (d Directory) ValidateDeliveryTarget(target gatewaydelivery.Target) (gatewaydelivery.Target, model.Evidence) {
	platform := model.NormalizePlatform(target.Platform)
	if target.IsOrigin || !target.IsExplicit || platform == "" || strings.EqualFold(platform, "local") || strings.TrimSpace(target.ChatID) == "" {
		return target, model.Evidence{}
	}
	entries, ok := d.Platforms[platform]
	if !ok || len(entries) == 0 {
		return target, model.Evidence{}
	}
	for _, entry := range entries {
		candidate := model.DeliveryTarget(platform, entry)
		if candidate.ChatID == strings.TrimSpace(target.ChatID) && candidate.ThreadID == strings.TrimSpace(target.ThreadID) {
			return target, model.Evidence{}
		}
	}
	return gatewaydelivery.Target{}, model.Evidence{Code: model.EvidenceChannelTargetStale, Platform: platform, Query: target.String()}
}

// LookupType returns the cached channel type for a platform target ID.
func (d Directory) LookupType(platform, id string) string {
	platform = model.NormalizePlatform(platform)
	id = strings.TrimSpace(id)
	for _, entry := range d.Platforms[platform] {
		if entry.ID == id {
			return strings.TrimSpace(entry.Type)
		}
	}
	return ""
}

package contract

import "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model/policy"

// EntryGuild returns the normalized display guild/server name carried by an
// entry. It is the shared guild identity contract for Discord display grouping
// and guild-qualified target resolution.
func EntryGuild(entry Entry) string {
	return policy.TrimText(entry.Guild)
}

// NormalizeGuildQuery returns the canonical guild/server key used when matching
// user-provided guild-qualified channel selectors.
func NormalizeGuildQuery(value string) string {
	return canonicalText(value)
}

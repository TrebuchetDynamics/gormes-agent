package entry

import entrycontract "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model/entry/contract"

// EntryGuild returns the normalized display guild/server name carried by an
// entry. It is the shared guild identity contract for Discord display grouping
// and guild-qualified target resolution.
func EntryGuild(entry Entry) string {
	return entrycontract.EntryGuild(entry)
}

// NormalizeGuildQuery returns the canonical guild/server key used when matching
// user-provided guild-qualified channel selectors.
func NormalizeGuildQuery(value string) string {
	return entrycontract.NormalizeGuildQuery(value)
}

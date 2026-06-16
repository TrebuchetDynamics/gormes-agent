package entry

import entrycontract "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model/entry/contract"

// Entry describes one platform target that can be selected by exact ID, human
// display name, guild-qualified name, or type/display lookup.
type Entry = entrycontract.Entry

// Evidence carries user-safe degraded-mode evidence without leaking local state paths.
type Evidence = entrycontract.Evidence

// NormalizePlatform returns the canonical platform key used by directory stores,
// refresh merges, and channel lookups.
func NormalizePlatform(platform string) string {
	return entrycontract.NormalizePlatform(platform)
}

// NormalizeEntry trims persisted target fields before merge and lookup.
func NormalizeEntry(entry Entry) Entry {
	return entrycontract.NormalizeEntry(entry)
}

// NormalizeQuery returns the canonical form used for human channel lookups.
func NormalizeQuery(value string) string {
	return entrycontract.NormalizeQuery(value)
}

// UpsertEntry replaces an existing entry with the same normalized ID or appends
// it when no cached target exists yet.
func UpsertEntry(entries []Entry, entry Entry) []Entry {
	return entrycontract.UpsertEntry(entries, entry)
}

// UpsertValidEntry normalizes and upserts a complete directory entry. It returns
// false without changing entries when the entry lacks the minimum cached-target
// contract used by refresh merges.
func UpsertValidEntry(entries []Entry, entry Entry) ([]Entry, bool) {
	return entrycontract.UpsertValidEntry(entries, entry)
}

package model

import sourcemodel "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model/source"

// Source is the minimal session-origin-shaped source needed to remember
// channel directory entries without depending on gateway runtime types.
type Source = sourcemodel.Source

// RememberedSourceEntry is the session-origin-shaped source record preserved
// for channel-directory refresh. Fields intentionally mirror Hermes session
// origin data plus enough metadata to produce Entry values.
type RememberedSourceEntry = sourcemodel.RememberedSourceEntry

// RememberedSourceLedger is the on-disk remembered-source ledger shape.
type RememberedSourceLedger = sourcemodel.RememberedSourceLedger

// EmptyRememberedSourceLedger returns a ledger with initialized platform buckets.
func EmptyRememberedSourceLedger() RememberedSourceLedger {
	return sourcemodel.EmptyRememberedSourceLedger()
}

// EnsureRememberedSourceLedger initializes the platform buckets after JSON decode.
func EnsureRememberedSourceLedger(ledger RememberedSourceLedger) RememberedSourceLedger {
	return sourcemodel.EnsureRememberedSourceLedger(ledger)
}

func RememberedSourceEntryFromSource(source Source) RememberedSourceEntry {
	return sourcemodel.RememberedSourceEntryFromSource(source)
}

func NormalizeRememberedSourceEntry(entry RememberedSourceEntry) RememberedSourceEntry {
	return sourcemodel.NormalizeRememberedSourceEntry(entry)
}

// UpsertRememberedSourceEntry replaces an existing remembered source with the
// same normalized ID or appends it when the session-discovered target is new.
// It returns false without changing entries when the source lacks the minimum
// remembered-source contract needed for refresh merges.
func UpsertRememberedSourceEntry(entries []RememberedSourceEntry, entry RememberedSourceEntry) ([]RememberedSourceEntry, bool) {
	return sourcemodel.UpsertRememberedSourceEntry(entries, entry)
}

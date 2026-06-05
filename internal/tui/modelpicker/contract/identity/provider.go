package identity

import "strings"

// ProviderIDEqual reports whether two provider IDs identify the same provider.
// Provider selection accepts case-insensitive IDs so slash-command input can
// preserve the operator's typed casing while still matching catalog entries.
func ProviderIDEqual(left, right string) bool {
	return strings.EqualFold(left, right)
}

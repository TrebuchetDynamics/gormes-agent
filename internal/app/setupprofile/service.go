package setupprofile

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ProfileID resolves the active setup profile ID from a GORMES_HOME-like path.
func ProfileID(home, defaultProfileID string) string {
	home = filepath.Clean(strings.TrimSpace(home))
	if home != "" && filepath.Base(filepath.Dir(home)) == "profiles" {
		if name := strings.ToLower(strings.TrimSpace(filepath.Base(home))); name != "" {
			return name
		}
	}
	return defaultProfileID
}

// RegistryPath returns the shared profile registry path under the base home.
func RegistryPath(baseHome string) string {
	return filepath.Join(baseHome, "config.toml")
}

// CredentialID returns the channel credential ID for a profile/channel pair.
func CredentialID(profileID, channelID string) string {
	profileID = strings.ToLower(strings.TrimSpace(profileID))
	channelID = strings.ToLower(strings.TrimSpace(channelID))
	channelID = strings.NewReplacer(".", "_", "/", "_", " ", "_").Replace(channelID)
	return profileID + "-" + channelID
}

// CompactStrings trims, drops blanks, and de-duplicates values while preserving order.
func CompactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

// Int64Strings formats int64 identifiers for profile channel allow-lists.
func Int64Strings(values []int64) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, fmt.Sprintf("%d", value))
	}
	return out
}

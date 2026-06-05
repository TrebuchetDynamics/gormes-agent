package whitelist

// WhitelistConfig represents a per-platform chat/channel/room whitelist.
// When Enabled is false, all chats are allowed (default-open behavior).
// When Enabled is true, only chats with IDs in the IDs slice are permitted.
type WhitelistConfig struct {
	Enabled bool
	IDs     []string
}

// WhitelistStatus carries runtime evidence for gateway status and doctor output.
type WhitelistStatus struct {
	ActiveCount  int
	SkippedCount int
	ParseError   string
}

// IsAllowed returns true if the whitelist is disabled (all chats allowed)
// or if chatID is present in the whitelist IDs.
func (wc WhitelistConfig) IsAllowed(chatID string) bool {
	if !wc.Enabled {
		return true
	}
	for _, id := range wc.IDs {
		if id == chatID {
			return true
		}
	}
	return false
}

// ParseWhitelistConfig parses a list of allowed IDs into a WhitelistConfig.
// Empty, whitespace-only, and duplicate entries are silently removed.
// An empty or nil input produces a disabled whitelist (Enabled=false).
func ParseWhitelistConfig(ids []string) WhitelistConfig {
	if len(ids) == 0 {
		return WhitelistConfig{}
	}
	seen := make(map[string]bool, len(ids))
	result := make([]string, 0, len(ids))
	for _, raw := range ids {
		id := trimSpaces(raw)
		if id == "" {
			continue
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		result = append(result, id)
	}
	if len(result) == 0 {
		return WhitelistConfig{}
	}
	return WhitelistConfig{Enabled: true, IDs: result}
}

func trimSpaces(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

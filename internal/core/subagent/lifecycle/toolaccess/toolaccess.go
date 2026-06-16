package toolaccess

import "strings"

// BlockedTools is the forward-looking list of tool names that subagents
// must not be allowed to invoke. Of these names, only delegate_task exists
// in the current Gormes tool surface; the others are placeholders for
// tools that will be added in later phases.
var BlockedTools = map[string]bool{
	"delegate_task": true,
	"clarify":       true,
	"memory":        true,
	"send_message":  true,
	"execute_code":  true,
}

func BlockedToolRequest(enabled []string) string {
	for _, rawName := range enabled {
		name := normalizeToolName(rawName)
		if name == "" {
			continue
		}
		if BlockedTools[name] {
			return name
		}
	}
	return ""
}

func ToolAllowlisted(enabled []string, name string) bool {
	name = normalizeToolName(name)
	if name == "" {
		return false
	}
	if len(enabled) == 0 {
		return !BlockedTools[name]
	}
	for _, allowed := range enabled {
		if normalizeToolName(allowed) == name {
			return !BlockedTools[name]
		}
	}
	return false
}

func normalizeToolName(name string) string {
	return strings.TrimSpace(name)
}

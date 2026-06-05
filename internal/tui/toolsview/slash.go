package toolsview

import (
	"fmt"
	"strings"
)

type Result struct {
	Changed        []string
	Unknown        []string
	MissingServers []string
	Reset          bool
}

type ConfigureFunc func(action string, names []string) (Result, error)

type SlashResult struct {
	Status   string
	Title    string
	Body     string
	Open     bool
	Fallback bool
}

func HandleSlash(input string, configure ConfigureFunc) SlashResult {
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) < 2 {
		return SlashResult{Fallback: true}
	}
	action := strings.ToLower(fields[1])
	if action != "enable" && action != "disable" {
		return SlashResult{Fallback: true}
	}
	names := append([]string(nil), fields[2:]...)
	if len(names) == 0 {
		return SlashResult{Status: Usage(action)}
	}
	if configure == nil {
		return SlashResult{Status: "tools: configuration unavailable"}
	}
	result, err := configure(action, names)
	if err != nil {
		return SlashResult{Status: "tools: " + err.Error()}
	}
	lines := Lines(action, result)
	if len(lines) == 0 {
		lines = []string{"tools: no changes"}
	}
	return SlashResult{Status: firstNonEmpty(lines), Title: "Tools", Body: strings.Join(lines, "\n"), Open: true}
}

func firstNonEmpty(values []string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func Usage(action string) string {
	return strings.Join([]string{
		fmt.Sprintf("usage: /tools %s <name> [name ...]", action),
		fmt.Sprintf("built-in toolset: /tools %s web", action),
		fmt.Sprintf("MCP tool: /tools %s github:create_issue", action),
	}, "\n")
}

func Lines(action string, result Result) []string {
	lines := make([]string, 0, 4)
	if len(result.Changed) > 0 {
		verb := "enabled"
		if action == "disable" {
			verb = "disabled"
		}
		lines = append(lines, fmt.Sprintf("%s: %s", verb, strings.Join(result.Changed, ", ")))
	}
	if len(result.Unknown) > 0 {
		lines = append(lines, "unknown toolsets: "+strings.Join(result.Unknown, ", "))
	}
	if len(result.MissingServers) > 0 {
		lines = append(lines, "missing MCP servers: "+strings.Join(result.MissingServers, ", "))
	}
	if result.Reset {
		lines = append(lines, "session reset. new tool configuration is active.")
	}
	return lines
}

package commandcatalog

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/redaction"
)

// Command is the platform-facing command/menu shape rendered in /commands.
type Command struct {
	Name        string
	Description string
}

// Request describes the channel-neutral data needed to render /commands.
type Request struct {
	Platform      string
	RawArgs       string
	BuiltinLines  []string
	SkillCommands []Command
}

// Render turns built-in and skill commands into a paginated operator reply.
func Render(req Request) string {
	requestedPage := 1
	if raw := strings.TrimSpace(req.RawArgs); raw != "" {
		page, ok := parseRequestedPage(raw)
		if !ok {
			return "Usage: /commands [page]"
		}
		requestedPage = page
	}

	entries := slices.Clone(req.BuiltinLines)
	skillCommands := sortedCommands(req.SkillCommands)
	var skillLines []string
	for _, cmd := range skillCommands {
		name := renderCommandCatalogValue(cmd.Name)
		if name == "" {
			continue
		}
		if !strings.HasPrefix(name, "/") {
			name = "/" + name
		}
		desc := renderCommandCatalogValue(cmd.Description)
		if desc == "" {
			desc = "Invoke skill"
		}
		skillLines = append(skillLines, fmt.Sprintf("`%s` -- %s", name, desc))
	}
	commandTotal := countCatalogCommandLines(entries) + len(skillLines)
	if len(skillLines) > 0 {
		entries = append(entries, "", "Skill commands:")
		entries = append(entries, skillLines...)
	}
	if commandTotal == 0 {
		return "No commands are available."
	}

	pageSize := pageSize(req.Platform)
	totalPages := (len(entries) + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	page := requestedPage
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > len(entries) {
		end = len(entries)
	}

	lines := []string{fmt.Sprintf("Available commands (%d total) — page %d/%d", commandTotal, page, totalPages), ""}
	lines = append(lines, entries[start:end]...)
	if totalPages > 1 {
		var nav []string
		if page > 1 {
			nav = append(nav, fmt.Sprintf("Prev: /commands %d", page-1))
		}
		if page < totalPages {
			nav = append(nav, fmt.Sprintf("Next: /commands %d", page+1))
		}
		if len(nav) > 0 {
			lines = append(lines, "", strings.Join(nav, " | "))
		}
	}
	if page != requestedPage {
		lines = append(lines, fmt.Sprintf("requested page %d out of range; showing page %d", requestedPage, page))
	}
	return strings.Join(lines, "\n")
}

func parseRequestedPage(raw string) (int, bool) {
	page, err := strconv.Atoi(raw)
	if err == nil {
		return page, true
	}
	for _, r := range raw {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	return int(^uint(0) >> 1), true
}

func countCatalogCommandLines(lines []string) int {
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

func renderCommandCatalogValue(value string) string {
	value = collapseRedactedCommandAssignments(redaction.RedactSecrets(value))
	replacer := strings.NewReplacer(
		"`", "'",
		"*", "'",
		"#", "＃",
	)
	return strings.Join(strings.Fields(replacer.Replace(value)), " ")
}

func collapseRedactedCommandAssignments(value string) string {
	replacer := strings.NewReplacer(
		"api_key=[redacted]", "[redacted]",
		"api-key=[redacted]", "[redacted]",
		"token=[redacted]", "[redacted]",
		"secret=[redacted]", "[redacted]",
		"password=[redacted]", "[redacted]",
	)
	return replacer.Replace(value)
}

func pageSize(platform string) int {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(platform)), "telegram") {
		return 15
	}
	return 20
}

func sortedCommands(commands []Command) []Command {
	out := slices.Clone(commands)
	slices.SortStableFunc(out, func(a, b Command) int {
		if byName := cmp.Compare(a.Name, b.Name); byName != 0 {
			return byName
		}
		return cmp.Compare(a.Description, b.Description)
	})
	return out
}

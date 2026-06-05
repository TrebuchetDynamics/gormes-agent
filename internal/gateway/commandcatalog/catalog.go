package commandcatalog

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
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
		page, err := strconv.Atoi(raw)
		if err != nil {
			return "Usage: /commands [page]"
		}
		requestedPage = page
	}

	entries := append([]string(nil), req.BuiltinLines...)
	skillCommands := sortedCommands(req.SkillCommands)
	if len(skillCommands) > 0 {
		entries = append(entries, "", "Skill commands:")
		for _, cmd := range skillCommands {
			name := strings.TrimSpace(cmd.Name)
			if name == "" {
				continue
			}
			if !strings.HasPrefix(name, "/") {
				name = "/" + name
			}
			desc := strings.TrimSpace(cmd.Description)
			if desc == "" {
				desc = "Invoke skill"
			}
			entries = append(entries, fmt.Sprintf("`%s` -- %s", name, desc))
		}
	}
	if len(entries) == 0 {
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

	lines := []string{fmt.Sprintf("Available commands (%d total) — page %d/%d", len(entries), page, totalPages), ""}
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

func pageSize(platform string) int {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(platform)), "telegram") {
		return 15
	}
	return 20
}

func sortedCommands(commands []Command) []Command {
	out := append([]Command(nil), commands...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Description < out[j].Description
	})
	return out
}

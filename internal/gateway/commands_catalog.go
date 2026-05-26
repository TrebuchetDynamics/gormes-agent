package gateway

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
)

type CommandsCatalogRequest struct {
	Platform      string
	RawArgs       string
	BuiltinLines  []string
	SkillCommands []PlatformCommand
}

func RenderCommandsCatalog(req CommandsCatalogRequest) string {
	requestedPage := 1
	if raw := strings.TrimSpace(req.RawArgs); raw != "" {
		page, err := strconv.Atoi(raw)
		if err != nil {
			return "Usage: /commands [page]"
		}
		requestedPage = page
	}

	entries := append([]string(nil), req.BuiltinLines...)
	skillCommands := sortedPlatformCommands(req.SkillCommands)
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

	pageSize := commandsCatalogPageSize(req.Platform)
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

func commandsCatalogPageSize(platform string) int {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(platform)), "telegram") {
		return 15
	}
	return 20
}

func (m *Manager) handleCommandsCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	platform := strings.TrimSpace(ev.Platform)
	if platform == "" && ch != nil {
		platform = ch.Name()
	}
	text := RenderCommandsCatalog(CommandsCatalogRequest{
		Platform:      platform,
		RawArgs:       strings.Join(commandArgs(ev.Text), " "),
		BuiltinLines:  GatewayHelpLines(),
		SkillCommands: m.enabledSkillPlatformCommands(ctx),
	})
	_, _ = m.sendWithHooks(ctx, ch, ev.ChatID, text)
}

func (m *Manager) enabledSkillPlatformCommands(ctx context.Context) []PlatformCommand {
	if m == nil || m.cfg.SkillRuntime == nil {
		return nil
	}
	commands, _, err := m.cfg.SkillRuntime.SkillSlashCommands(ctx, skills.RuntimeOptions{})
	if err != nil {
		if m.log != nil {
			m.log.Warn("gateway: skill command catalog scan failed", "err", err)
		}
		return nil
	}
	out := make([]PlatformCommand, 0, len(commands))
	seen := map[string]struct{}{}
	for _, command := range commands {
		name := strings.TrimPrefix(strings.TrimSpace(command.Command), "/")
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, PlatformCommand{Name: name, Description: command.Description})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Description < out[j].Description
	})
	return out
}

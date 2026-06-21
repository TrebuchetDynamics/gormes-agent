package gateway

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// handleToolsCommand implements /tools [list] — lists active tools grouped by
// toolset, mirroring Hermes show_tools(). enable/disable subcommands are
// routed to TUI config; in the gateway we report that config path.
func (m *Manager) handleToolsCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	body := strings.TrimSpace(ev.Text)
	// Strip the "/tools" prefix if present (gateways normalize ev.Text to the
	// subcommand portion, but some callers pass the full slash command).
	if after, ok := strings.CutPrefix(body, "/tools"); ok {
		body = strings.TrimSpace(after)
	}
	sub := strings.ToLower(strings.SplitN(body, " ", 2)[0])

	switch sub {
	case "", "list":
		_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, m.formatToolsList())
	case "enable", "disable":
		_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID,
			fmt.Sprintf("/tools %s is managed via the Gormes config (platform_toolsets). "+
				"Use `gormes tools %s <toolset>` from the CLI or the /tools TUI panel.", sub, sub))
	default:
		_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID,
			"Usage: /tools [list|enable|disable]")
	}
}

// formatToolsList renders the active tool registry as a grouped text summary.
func (m *Manager) formatToolsList() string {
	reg := m.cfg.ToolRegistry
	if reg == nil {
		return "No tool registry available."
	}
	descs := reg.Descriptors()
	if len(descs) == 0 {
		return "No tools are currently active."
	}

	// Group by toolset prefix (first word of name before underscore).
	groups := map[string][]string{}
	for _, d := range descs {
		group := toolsetGroup(d.Name)
		summary := d.Name
		if d.Description != "" {
			line := strings.SplitN(d.Description, "\n", 2)[0]
			if idx := strings.Index(line, ". "); idx >= 0 {
				line = line[:idx+1]
			}
			if len(line) > 80 {
				line = line[:77] + "..."
			}
			summary = fmt.Sprintf("%-22s %s", d.Name, line)
		}
		groups[group] = append(groups[group], summary)
	}

	groupNames := make([]string, 0, len(groups))
	for k := range groups {
		groupNames = append(groupNames, k)
	}
	sort.Strings(groupNames)

	var sb strings.Builder
	fmt.Fprintf(&sb, "Active tools (%d total):\n", len(descs))
	for _, g := range groupNames {
		fmt.Fprintf(&sb, "\n[%s]\n", g)
		sort.Strings(groups[g])
		for _, line := range groups[g] {
			fmt.Fprintf(&sb, "  %s\n", line)
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

// toolsetGroup infers a display grouping from a tool name.
// Uses the first underscore-delimited component as the group;
// falls back to "core" for names without underscores.
func toolsetGroup(name string) string {
	if idx := strings.Index(name, "_"); idx > 0 {
		return name[:idx]
	}
	return "core"
}

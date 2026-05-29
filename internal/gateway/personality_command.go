package gateway

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// handlePersonalityCommand handles /personality subcommands (list, switch, none).
func (m *Manager) handlePersonalityCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	personalities := m.loadPersonalities()
	arg := parsePersonalityArg(ev.Text)

	// /personality — list available personalities
	if arg == "" {
		active := m.activePersonality()
		lines := []string{"**Personalities:**"}
		if active != "" {
			lines = append(lines, fmt.Sprintf("Active: **%s**", active))
		} else {
			lines = append(lines, "Active: *(none)*")
		}
		if len(personalities) > 0 {
			names := make([]string, 0, len(personalities))
			for name := range personalities {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				desc := truncatePersonalityDesc(personalities[name], 60)
				lines = append(lines, fmt.Sprintf("  • `/personality %s` — %s", name, desc))
			}
		} else {
			lines = append(lines, "(no personalities configured)")
		}
		lines = append(lines, "", "Usage: `/personality <name>` or `/personality none` to clear")
		_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, strings.Join(lines, "\n"))
		return
	}

	// /personality none — clear
	if strings.ToLower(strings.TrimSpace(arg)) == "none" {
		m.setActivePersonality("")
		_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, "Personality cleared.")
		return
	}

	// /personality <name> — switch
	name := strings.ToLower(strings.TrimSpace(arg))
	prompt, ok := personalities[name]
	if !ok {
		known := make([]string, 0, len(personalities))
		for n := range personalities {
			known = append(known, n)
		}
		sort.Strings(known)
		var hint string
		if len(known) > 0 {
			hint = " Available: " + strings.Join(known, ", ")
		}
		_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID,
			fmt.Sprintf("Unknown personality %q.%s", name, hint))
		return
	}
	m.setActivePersonality(name)
	_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID,
		fmt.Sprintf("Personality set to **%s**.", name))
	_ = prompt // used at prompt assembly time
}

// loadPersonalities returns the configured personality map. Defaults to nil.
func (m *Manager) loadPersonalities() map[string]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.personalityPrompts
}

// activePersonality returns the currently active personality name.
func (m *Manager) activePersonality() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.activePersonalityName
}

// setActivePersonality sets the active personality name.
func (m *Manager) setActivePersonality(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activePersonalityName = name
}

func parsePersonalityArg(text string) string {
	body := strings.TrimSpace(text)
	if body == "" {
		return ""
	}
	fields := strings.Fields(body)
	if len(fields) == 0 || slashCommandName(fields[0]) != "personality" {
		return body
	}
	idx := strings.Index(body, fields[0])
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(body[idx+len(fields[0]):])
}

func truncatePersonalityDesc(s string, max int) string {
	if len([]rune(s)) <= max {
		return s
	}
	return string([]rune(s)[:max]) + "…"
}

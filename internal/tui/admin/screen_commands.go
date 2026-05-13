package admin

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// CommandEntry is one command surfaced in the unified admin TUI command
// catalog. cmd/gormes builds these from the live Cobra command tree so this
// package stays independent of Cobra and command implementations.
type CommandEntry struct {
	Path  string
	Use   string
	Short string
}

// CommandsScreen lists the Gormes CLI command tree inside the admin TUI.
type CommandsScreen struct {
	entries  []CommandEntry
	selected int
}

// NewCommandsScreen returns the command catalog tab.
func NewCommandsScreen(entries []CommandEntry) *CommandsScreen {
	entries = cloneCommandEntries(entries)
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})
	return &CommandsScreen{entries: entries}
}

func (s *CommandsScreen) Title() string { return "Commands" }

func (s *CommandsScreen) Init() tea.Cmd { return nil }

func (s *CommandsScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if s.selected > 0 {
				s.selected--
			}
		case "down", "j":
			if s.selected < len(s.entries)-1 {
				s.selected++
			}
		case "home":
			s.selected = 0
		case "end":
			if len(s.entries) > 0 {
				s.selected = len(s.entries) - 1
			}
		}
	}
	return s, nil
}

func (s *CommandsScreen) View() string {
	var b strings.Builder
	fmt.Fprintf(&b, "CLI commands (%d commands)\n", len(s.entries))
	if len(s.entries) == 0 {
		b.WriteString("No commands discovered.\n")
		return b.String()
	}
	start, end := commandWindow(s.selected, len(s.entries), 12)
	for i := start; i < end; i++ {
		entry := s.entries[i]
		marker := " "
		if i == s.selected {
			marker = ">"
		}
		fmt.Fprintf(&b, "%s %s", marker, entry.Use)
		if entry.Short != "" {
			fmt.Fprintf(&b, "  - %s", entry.Short)
		}
		b.WriteByte('\n')
	}
	entry := s.entries[s.selected]
	b.WriteByte('\n')
	fmt.Fprintf(&b, "Selected: %s\n", entry.Use)
	if entry.Short != "" {
		fmt.Fprintf(&b, "%s\n", entry.Short)
	}
	b.WriteString("Run this command from your shell; mutating commands stay explicit.\n")
	return b.String()
}

func (s *CommandsScreen) ShortHelp() []KeyHelp {
	return []KeyHelp{
		{Keys: []string{"up", "down"}, Description: "select command"},
		{Keys: []string{"home", "end"}, Description: "jump to start/end"},
	}
}

func commandWindow(selected, total, size int) (int, int) {
	if total <= size {
		return 0, total
	}
	start := selected - size/2
	if start < 0 {
		start = 0
	}
	if start+size > total {
		start = total - size
	}
	return start, start + size
}

func cloneCommandEntries(entries []CommandEntry) []CommandEntry {
	out := make([]CommandEntry, 0, len(entries))
	for _, entry := range entries {
		entry.Path = strings.TrimSpace(entry.Path)
		entry.Use = strings.TrimSpace(entry.Use)
		entry.Short = strings.TrimSpace(entry.Short)
		if entry.Path == "" && entry.Use == "" {
			continue
		}
		if entry.Use == "" {
			entry.Use = "gormes " + entry.Path
		}
		out = append(out, entry)
	}
	return out
}

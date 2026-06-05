package commands

import (
	"fmt"
	"sort"
	"strings"

	admincontracts "github.com/TrebuchetDynamics/gormes-agent/internal/tui/admin/contracts"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/admin/navigation"
	tea "github.com/charmbracelet/bubbletea"
)

type CommandEntry = admincontracts.CommandEntry
type CommandRunner = admincontracts.CommandRunner
type CommandRunResult = admincontracts.CommandRunResult

type commandRunFinishedMsg struct {
	result CommandRunResult
}

// CommandsScreen lists the Gormes CLI command tree inside the admin TUI.
type Screen struct {
	entries   []CommandEntry
	selected  int
	runner    CommandRunner
	running   string
	result    *CommandRunResult
	query     string
	searching bool
}

// CommandScreenOption configures the command catalog tab.
type Option func(*Screen)

// WithCommandRunner lets the command catalog execute safe commands inline.
func WithCommandRunner(runner CommandRunner) Option {
	return func(s *Screen) {
		s.runner = runner
	}
}

// NewCommandsScreen returns the command catalog tab.
func NewScreen(entries []CommandEntry, opts ...Option) *Screen {
	entries = cloneCommandEntries(entries)
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})
	s := &Screen{entries: entries}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Screen) Title() string { return "Commands" }

func (s *Screen) Init() tea.Cmd { return nil }

func (s *Screen) CapturesKey(msg tea.KeyMsg) bool {
	if !s.searching {
		return false
	}
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyTab, tea.KeyShiftTab:
		return false
	default:
		return true
	}
}

func (s *Screen) Update(msg tea.Msg) (admincontracts.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case commandRunFinishedMsg:
		result := msg.result
		if result.RunLabel == "" {
			result.RunLabel = s.running
		}
		s.running = ""
		s.result = &result
	case tea.KeyMsg:
		if s.searching {
			return s.updateSearchKey(msg)
		}
		if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && msg.Runes[0] == '/' {
			s.searching = true
			s.query = ""
			s.selected = 0
			s.result = nil
			return s, nil
		}
		switch msg.String() {
		case "up", "k":
			s.moveSelected(-1)
		case "down", "j":
			s.moveSelected(1)
		case "home":
			s.selected = 0
		case "end":
			s.selectEnd()
		case "enter":
			return s, s.runSelectedCmd()
		}
	}
	return s, nil
}

func (s *Screen) updateSearchKey(msg tea.KeyMsg) (admincontracts.Screen, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		s.searching = false
		s.query = ""
		s.selected = 0
		s.result = nil
	case tea.KeyBackspace, tea.KeyCtrlH:
		s.removeQueryRune()
	case tea.KeyUp:
		s.moveSelected(-1)
	case tea.KeyDown:
		s.moveSelected(1)
	case tea.KeyHome:
		s.selected = 0
	case tea.KeyEnd:
		s.selectEnd()
	case tea.KeyEnter:
		return s, s.runSelectedCmd()
	case tea.KeySpace:
		s.appendQuery(" ")
	case tea.KeyRunes:
		s.appendQuery(string(msg.Runes))
	}
	return s, nil
}

func (s *Screen) View() string {
	var b strings.Builder
	visible := s.visibleEntries()
	if s.searching || s.query != "" {
		fmt.Fprintf(&b, "CLI commands (%d of %d commands)\n", len(visible), len(s.entries))
		fmt.Fprintf(&b, "Search: %s\n", s.query)
	} else {
		fmt.Fprintf(&b, "CLI commands (%d commands)\n", len(s.entries))
	}
	if len(s.entries) == 0 {
		b.WriteString("No commands discovered.\n")
		return b.String()
	}
	if len(visible) == 0 {
		fmt.Fprintf(&b, "No commands match %q.\n", s.query)
		return b.String()
	}
	selected := navigation.ClampIndex(s.selected, len(visible))
	start, end := navigation.Window(selected, len(visible), 12)
	for i := start; i < end; i++ {
		entry := visible[i]
		marker := " "
		if i == selected {
			marker = ">"
		}
		fmt.Fprintf(&b, "%s %s", marker, entry.Use)
		if entry.Short != "" {
			fmt.Fprintf(&b, "  - %s", entry.Short)
		}
		b.WriteByte('\n')
	}
	entry := visible[selected]
	b.WriteByte('\n')
	fmt.Fprintf(&b, "Selected: %s\n", entry.Use)
	if entry.Short != "" {
		fmt.Fprintf(&b, "%s\n", entry.Short)
	}
	if entry.Runnable {
		fmt.Fprintf(&b, "Enter: run %s\n", commandRunLabel(entry))
	} else {
		b.WriteString("Run this command from your shell; mutating commands stay explicit.\n")
	}
	if s.running != "" {
		fmt.Fprintf(&b, "\nRunning: %s\n", s.running)
	}
	if s.result != nil {
		fmt.Fprintf(&b, "\n")
		if s.result.Error != "" {
			fmt.Fprintf(&b, "Command failed: %s\n%s\n", s.result.RunLabel, s.result.Error)
		} else {
			fmt.Fprintf(&b, "Command output: %s\n", s.result.RunLabel)
		}
		if strings.TrimSpace(s.result.Output) != "" {
			b.WriteString(s.result.Output)
			if !strings.HasSuffix(s.result.Output, "\n") {
				b.WriteByte('\n')
			}
		}
	}
	return b.String()
}

func (s *Screen) ShortHelp() []admincontracts.KeyHelp {
	return []admincontracts.KeyHelp{
		{Keys: []string{"enter"}, Description: "run safe command"},
		{Keys: []string{"/"}, Description: "search commands"},
		{Keys: []string{"esc"}, Description: "clear search"},
		{Keys: []string{"up", "down"}, Description: "select command"},
		{Keys: []string{"home", "end"}, Description: "jump to start/end"},
	}
}

func (s *Screen) runSelectedCmd() tea.Cmd {
	entry, ok := s.selectedEntry()
	if !ok {
		return nil
	}
	label := commandRunLabel(entry)
	if !entry.Runnable || s.runner == nil {
		s.result = &CommandRunResult{
			RunLabel: label,
			Error:    "Command is not runnable inside gormes admin; run it from your shell so mutating commands stay explicit.",
		}
		return nil
	}
	s.running = label
	s.result = nil
	return func() tea.Msg {
		result := s.runner(entry)
		if result.RunLabel == "" {
			result.RunLabel = label
		}
		return commandRunFinishedMsg{result: result}
	}
}

func (s *Screen) selectedEntry() (CommandEntry, bool) {
	visible := s.visibleEntries()
	if len(visible) == 0 {
		return CommandEntry{}, false
	}
	s.clampSelected(len(visible))
	return visible[s.selected], true
}

func (s *Screen) visibleEntries() []CommandEntry {
	query := strings.TrimSpace(s.query)
	if query == "" {
		return s.entries
	}
	out := make([]CommandEntry, 0, len(s.entries))
	for _, entry := range s.entries {
		if matchesCommandEntry(entry, query) {
			out = append(out, entry)
		}
	}
	return out
}

func matchesCommandEntry(entry CommandEntry, query string) bool {
	query = strings.ToLower(query)
	return strings.Contains(strings.ToLower(entry.Path), query) ||
		strings.Contains(strings.ToLower(entry.Use), query) ||
		strings.Contains(strings.ToLower(entry.Short), query) ||
		strings.Contains(strings.ToLower(entry.RunLabel), query)
}

func (s *Screen) appendQuery(text string) {
	if text == "" {
		return
	}
	s.query += text
	s.selected = 0
	s.result = nil
}

func (s *Screen) removeQueryRune() {
	if s.query == "" {
		return
	}
	runes := []rune(s.query)
	s.query = string(runes[:len(runes)-1])
	s.selected = 0
	s.result = nil
}

func (s *Screen) moveSelected(delta int) {
	s.selected = navigation.MoveIndex(s.selected, len(s.visibleEntries()), delta)
}

func (s *Screen) selectEnd() {
	s.selected = navigation.ClampIndex(len(s.visibleEntries())-1, len(s.visibleEntries()))
}

func (s *Screen) clampSelected(total int) {
	s.selected = navigation.ClampIndex(s.selected, total)
}

func commandRunLabel(entry CommandEntry) string {
	if entry.RunLabel != "" {
		return entry.RunLabel
	}
	if entry.Use != "" {
		return entry.Use
	}
	return "gormes " + entry.Path
}

// CloneEntries returns normalized command catalog entries without sharing the
// caller's backing slice.
func CloneEntries(entries []CommandEntry) []CommandEntry {
	return cloneCommandEntries(entries)
}

func cloneCommandEntries(entries []CommandEntry) []CommandEntry {
	out := make([]CommandEntry, 0, len(entries))
	for _, entry := range entries {
		entry.Path = strings.TrimSpace(entry.Path)
		entry.Use = strings.TrimSpace(entry.Use)
		entry.Short = strings.TrimSpace(entry.Short)
		entry.RunLabel = strings.TrimSpace(entry.RunLabel)
		if entry.Path == "" && entry.Use == "" {
			continue
		}
		if entry.Use == "" {
			entry.Use = "gormes " + entry.Path
		}
		if entry.Runnable && entry.RunLabel == "" {
			entry.RunLabel = entry.Use
		}
		out = append(out, entry)
	}
	return out
}

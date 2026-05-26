// Package admin hosts the unified Gormes admin TUI: a Bubble Tea shell
// that owns tab navigation, the status bar, and global keybindings while
// delegating content to per-screen Bubble Tea sub-models.
//
// The shell is invoked by `gormes admin`. Concrete screens (Setup health,
// Chat, Agents, ...) live in sibling files and register with New.
package admin

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	basetui "github.com/TrebuchetDynamics/gormes-agent/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"golang.org/x/term"
)

// ErrRequiresTTY is returned by Run when the configured input is not a
// terminal. Callers translate this into admin_tui_requires_tty operator
// evidence so automation does not hang on a piped stdin.
var ErrRequiresTTY = errors.New("admin_tui_requires_tty")

type adminShellStyles struct {
	ActiveTab   lipgloss.Style
	InactiveTab lipgloss.Style
	Separator   lipgloss.Style
	Status      lipgloss.Style
	HelpHeader  lipgloss.Style
	HelpSection lipgloss.Style
	HelpKey     lipgloss.Style
	HelpText    lipgloss.Style
}

func adminShellStylesForSkin(skin basetui.HermesSkin) adminShellStyles {
	skin = basetui.NormalizeStyleSkin(skin)
	shared := basetui.SkinStylesFor(skin)
	return adminShellStyles{
		ActiveTab: shared.Status.
			Foreground(lipgloss.Color(skin.Colors.StatusBarBackground)).
			Background(lipgloss.Color(skin.Colors.UIAcent)).
			Bold(true),
		InactiveTab: shared.Dim,
		Separator:   shared.Separator,
		Status:      shared.Status,
		HelpHeader:  shared.Title,
		HelpSection: shared.Label,
		HelpKey:     shared.Accent,
		HelpText:    shared.Text,
	}
}

// Shell is the Bubble Tea root model for the admin TUI. Tests drive it
// directly via teatest.NewTestModel; production callers use Run.
type Shell struct {
	mu       sync.Mutex
	screens  []Screen
	active   int
	previous int

	width  int
	height int

	helpOpen bool
	done     bool
	err      error
}

// New constructs a Shell with the supplied ordered screens. At least one
// screen must be supplied; the first screen is initially active.
func New(screens ...Screen) *Shell {
	return &Shell{screens: screens}
}

// ActiveIndex returns the index of the currently focused screen.
// Safe to call from outside the Bubble Tea event loop (used by tests
// to poll for navigation effects after sending key messages).
func (s *Shell) ActiveIndex() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active
}

// Err returns any terminal error recorded by the shell (currently only
// reserved for future use; aborts via Ctrl-C / 'q' are clean exits).
func (s *Shell) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

// Init starts the first screen.
func (s *Shell) Init() tea.Cmd {
	if len(s.screens) == 0 {
		return tea.Quit
	}
	return s.screens[0].Init()
}

// Update dispatches messages: global keys (tab navigation, quit, help)
// are intercepted by the shell; everything else is forwarded to the
// active screen.
func (s *Shell) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.mu.Lock()
		s.width, s.height = msg.Width, msg.Height
		s.mu.Unlock()
		return s.forwardToActive(tea.WindowSizeMsg{Width: msg.Width, Height: adminShellBodyHeight(msg.Height)})
	case tea.KeyMsg:
		if s.activeScreenCapturesKey(msg) {
			return s.forwardToActive(msg)
		}
		if cmd, handled := s.handleGlobalKey(msg); handled {
			return s, cmd
		}
	}
	return s.forwardToActive(msg)
}

func (s *Shell) activeScreenCapturesKey(msg tea.KeyMsg) bool {
	s.mu.Lock()
	if s.active < 0 || s.active >= len(s.screens) {
		s.mu.Unlock()
		return false
	}
	current := s.screens[s.active]
	s.mu.Unlock()

	capturing, ok := current.(KeyCapturingScreen)
	return ok && capturing.CapturesKey(msg)
}

func (s *Shell) handleGlobalKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch msg.Type {
	case tea.KeyCtrlC:
		s.mu.Lock()
		s.done = true
		s.mu.Unlock()
		return tea.Quit, true
	case tea.KeyTab:
		return s.advance(1), true
	case tea.KeyShiftTab:
		return s.advance(-1), true
	case tea.KeyRunes:
		if len(msg.Runes) == 1 {
			r := msg.Runes[0]
			if r >= '1' && r <= '9' {
				if cmd, ok := s.jumpTo(int(r - '1')); ok {
					return cmd, true
				}
			}
			if r == 'c' && !s.isActiveTitle("Chat") {
				if cmd, ok := s.jumpToTitle("Chat"); ok {
					return cmd, true
				}
			}
			if r == '?' {
				s.toggleHelp()
				return nil, true
			}
			if r == 'q' {
				s.mu.Lock()
				s.done = true
				s.mu.Unlock()
				return tea.Quit, true
			}
		}
	case tea.KeyEscape:
		if cmd, ok := s.returnToPreviousFrom("Chat"); ok {
			return cmd, true
		}
	}
	return nil, false
}

func (s *Shell) toggleHelp() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.helpOpen = !s.helpOpen
}

func (s *Shell) jumpTo(idx int) (tea.Cmd, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if idx < 0 || idx >= len(s.screens) {
		return nil, false
	}
	return s.setActiveLocked(idx), true
}

func (s *Shell) jumpToTitle(title string) (tea.Cmd, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, sc := range s.screens {
		if strings.EqualFold(sc.Title(), title) {
			return s.setActiveLocked(i), true
		}
	}
	return nil, false
}

func (s *Shell) isActiveTitle(title string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active >= 0 && s.active < len(s.screens) && strings.EqualFold(s.screens[s.active].Title(), title)
}

func (s *Shell) returnToPreviousFrom(title string) (tea.Cmd, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active < 0 || s.active >= len(s.screens) || !strings.EqualFold(s.screens[s.active].Title(), title) {
		return nil, false
	}
	if s.previous < 0 || s.previous >= len(s.screens) || s.previous == s.active {
		return nil, false
	}
	return s.setActiveLocked(s.previous), true
}

func (s *Shell) advance(delta int) tea.Cmd {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.screens) == 0 {
		return nil
	}
	n := len(s.screens)
	next := ((s.active+delta)%n + n) % n
	return s.setActiveLocked(next)
}

func (s *Shell) setActiveLocked(idx int) tea.Cmd {
	if idx < 0 || idx >= len(s.screens) || idx == s.active {
		return nil
	}
	s.previous = s.active
	s.active = idx
	cmds := []tea.Cmd{s.screens[s.active].Init()}
	if s.width > 0 || s.height > 0 {
		next, resizeCmd := s.screens[s.active].Update(tea.WindowSizeMsg{Width: s.width, Height: adminShellBodyHeight(s.height)})
		s.screens[s.active] = next
		cmds = append(cmds, resizeCmd)
	}
	return tea.Batch(cmds...)
}

func (s *Shell) forwardToActive(msg tea.Msg) (tea.Model, tea.Cmd) {
	s.mu.Lock()
	if s.active >= len(s.screens) {
		s.mu.Unlock()
		return s, nil
	}
	current := s.screens[s.active]
	idx := s.active
	s.mu.Unlock()

	next, cmd := current.Update(msg)

	s.mu.Lock()
	s.screens[idx] = next
	s.mu.Unlock()
	return s, cmd
}

// View renders the tab bar, active screen body, and status bar.
func (s *Shell) View() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.done {
		return ""
	}
	if len(s.screens) == 0 {
		return "no screens registered\n"
	}

	var b strings.Builder
	b.WriteString(s.renderTabBarLocked())
	b.WriteString("\n")
	if s.helpOpen {
		b.WriteString(s.renderHelpOverlayLocked())
	} else {
		b.WriteString(s.screens[s.active].View())
	}
	b.WriteString("\n")
	b.WriteString(s.renderStatusBarLocked())
	return b.String()
}

func (s *Shell) renderHelpOverlayLocked() string {
	styles := adminShellStylesForSkin(basetui.DefaultHermesSkin())
	var b strings.Builder
	b.WriteString(styles.HelpHeader.Render("Admin help (? to close)"))
	b.WriteByte('\n')
	for _, sc := range s.screens {
		entries := sc.ShortHelp()
		if len(entries) == 0 {
			continue
		}
		fmt.Fprintf(&b, "  %s\n", styles.HelpSection.Render(sc.Title()+":"))
		for _, e := range entries {
			keys := styles.HelpKey.Render(strings.Join(e.Keys, "/"))
			desc := styles.HelpText.Render(e.Description)
			fmt.Fprintf(&b, "    %s  %s\n", keys, desc)
		}
	}
	return clampAdminShellBlock(b.String(), s.width, adminShellBodyHeight(s.height))
}

func (s *Shell) renderTabBarLocked() string {
	styles := adminShellStylesForSkin(basetui.DefaultHermesSkin())
	parts := make([]string, 0, len(s.screens))
	for i, sc := range s.screens {
		label := fmt.Sprintf("%d %s", i+1, sc.Title())
		if i == s.active {
			label = styles.ActiveTab.Render("[" + label + "]")
		} else {
			label = styles.InactiveTab.Render(" " + label + " ")
		}
		parts = append(parts, label)
	}
	return trimAdminShellLine(strings.Join(parts, styles.Separator.Render(" ")), s.width)
}

func (s *Shell) renderStatusBarLocked() string {
	styles := adminShellStylesForSkin(basetui.DefaultHermesSkin())
	return styles.Status.Render(trimAdminShellLine("⚕ Gormes · tab/shift+tab cycle · q quit · ? help", s.width))
}

func adminShellBodyHeight(total int) int {
	if total <= 0 {
		return 0
	}
	// Shell View adds one tab-bar line and one status-bar line around the active
	// screen body. Give screens the remaining row budget so real terminal output
	// stays within the operator's current height.
	return max(1, total-2)
}

func clampAdminShellBlock(text string, width, height int) string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	trimmed := false
	for i, line := range lines {
		if width > 0 && lipgloss.Width(line) > width {
			trimmed = true
		}
		lines[i] = trimAdminShellLine(line, width)
	}
	if height <= 0 {
		return strings.Join(lines, "\n")
	}
	if trimmed && len(lines) >= height && height > 2 {
		lines[height-1] = trimAdminShellLine("… omitted; resize for full help", width)
		lines = lines[:height]
	}
	if len(lines) <= height {
		return strings.Join(lines, "\n")
	}
	if height <= 2 {
		return trimAdminShellLine("terminal too small; resize", width)
	}
	marker := trimAdminShellLine("… omitted; resize for full help", width)
	tailCount := 1
	headCount := height - tailCount - 1
	if headCount < 1 {
		headCount = 1
	}
	out := append([]string(nil), lines[:headCount]...)
	out = append(out, marker)
	out = append(out, lines[len(lines)-tailCount:]...)
	return strings.Join(out, "\n")
}

func trimAdminShellLine(text string, width int) string {
	if width <= 0 || lipgloss.Width(text) <= width {
		return text
	}
	if width == 1 {
		return "…"
	}
	return ansi.Truncate(strings.TrimRight(text, " \t"), width, "…")
}

// Run is the production-facing entry. It refuses to start when in is not
// a terminal and returns ErrRequiresTTY so callers can emit a typed
// admin_tui_requires_tty evidence string instead of hanging on a piped
// stdin. Tests should use New + teatest.NewTestModel directly.
func Run(in *os.File, out io.Writer, screens ...Screen) error {
	if in == nil || !term.IsTerminal(int(in.Fd())) {
		return ErrRequiresTTY
	}
	shell := New(screens...)
	p := tea.NewProgram(shell, tea.WithInput(in), tea.WithOutput(out))
	if _, err := p.Run(); err != nil {
		return err
	}
	return shell.Err()
}

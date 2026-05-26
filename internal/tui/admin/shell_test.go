package admin

import (
	"strings"
	"testing"
	"time"

	basetui "github.com/TrebuchetDynamics/gormes-agent/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
)

// stubScreen is the minimal Screen implementation used by the shell tests.
// It records its initialization and exposes its title via the interface.
type stubScreen struct {
	name string
	help []KeyHelp
}

func (s *stubScreen) Title() string                    { return s.name }
func (s *stubScreen) Init() tea.Cmd                    { return nil }
func (s *stubScreen) Update(tea.Msg) (Screen, tea.Cmd) { return s, nil }
func (s *stubScreen) View() string                     { return s.name + " body" }
func (s *stubScreen) ShortHelp() []KeyHelp             { return s.help }

// TestAdminShell_TabBarCyclesScreens registers two stub screens, drives the
// shell through tab and shift-tab, and asserts the active screen index moves
// forward, wraps, and reverses.
func TestAdminShell_TabBarCyclesScreens(t *testing.T) {
	shell := New(
		&stubScreen{name: "Setup"},
		&stubScreen{name: "Agents"},
	)
	tm := teatest.NewTestModel(t, shell, teatest.WithInitialTermSize(80, 24))

	if got := shell.ActiveIndex(); got != 0 {
		t.Fatalf("initial ActiveIndex = %d, want 0", got)
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	waitForActive(t, shell, 1)

	// Tab again wraps back to 0.
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	waitForActive(t, shell, 0)

	// Shift-tab reverses (and wraps).
	tm.Send(tea.KeyMsg{Type: tea.KeyShiftTab})
	waitForActive(t, shell, 1)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

// TestAdminShell_DigitKeysJumpToScreen registers three stub screens and
// asserts that pressing '1', '2', '3' jumps the active index directly to
// each corresponding screen.
func TestAdminShell_DigitKeysJumpToScreen(t *testing.T) {
	shell := New(
		&stubScreen{name: "Setup"},
		&stubScreen{name: "Chat"},
		&stubScreen{name: "Agents"},
	)
	tm := teatest.NewTestModel(t, shell, teatest.WithInitialTermSize(80, 24))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	waitForActive(t, shell, 1)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	waitForActive(t, shell, 2)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	waitForActive(t, shell, 0)

	// Out-of-range digits are ignored (screen count is 3).
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'9'}})
	waitForActive(t, shell, 0)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

// TestAdminShell_HelpOverlayShowsRegisteredKeybindings asserts that pressing
// '?' toggles a help overlay containing the merged ShortHelp entries from
// every registered screen.
func TestAdminShell_GlobalChromeUsesSkinStyles(t *testing.T) {
	skin := basetui.DefaultHermesSkin()
	shared := basetui.SkinStylesFor(skin)
	styles := adminShellStylesForSkin(skin)

	if got, want := styles.ActiveTab.GetForeground(), lipgloss.Color(skin.Colors.StatusBarBackground); got != want {
		t.Fatalf("active tab foreground = %v, want %v", got, want)
	}
	if got, want := styles.ActiveTab.GetBackground(), lipgloss.Color(skin.Colors.UIAcent); got != want {
		t.Fatalf("active tab background = %v, want %v", got, want)
	}
	if got, want := styles.Status.GetForeground(), shared.Status.GetForeground(); got != want {
		t.Fatalf("status foreground = %v, want shared %v", got, want)
	}
	if got, want := styles.HelpKey.GetForeground(), shared.Accent.GetForeground(); got != want {
		t.Fatalf("help key foreground = %v, want shared %v", got, want)
	}

	shell := New(&stubScreen{name: "Setup"}, &stubScreen{name: "Agents"})
	updated, _ := shell.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	shell = updated.(*Shell)
	view := shell.View()
	for _, want := range []string{"[1 Setup]", " 2 Agents ", "⚕ Gormes", "tab/shift+tab cycle", "? help"} {
		if !strings.Contains(view, want) {
			t.Fatalf("styled admin chrome missing %q:\n%s", want, view)
		}
	}
}

func TestAdminShell_HelpOverlayShowsRegisteredKeybindings(t *testing.T) {
	shell := New(
		&stubScreen{
			name: "Setup",
			help: []KeyHelp{{Keys: []string{"r"}, Description: "refresh checks"}},
		},
		&stubScreen{
			name: "Agents",
			help: []KeyHelp{
				{Keys: []string{"n"}, Description: "spawn agent"},
				{Keys: []string{"b"}, Description: "bind agent"},
			},
		},
	)

	next, _ := shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	shell = next.(*Shell)
	got := shell.View()
	for _, want := range []string{"Admin help", "refresh checks", "spawn agent", "bind agent"} {
		if !strings.Contains(got, want) {
			t.Fatalf("help overlay missing %q:\n%s", want, got)
		}
	}

	// Press '?' again to close the help overlay; the body returns to the
	// active screen's normal View().
	next, _ = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	shell = next.(*Shell)
	got = shell.View()
	if !strings.Contains(got, "Setup body") || strings.Contains(got, "Admin help") {
		t.Fatalf("help overlay did not close back to active screen:\n%s", got)
	}
}

func TestAdminShell_HelpOverlayBoundsCrampedSetupTerminal(t *testing.T) {
	long := strings.Repeat("x", 160)
	shell := New(
		&stubScreen{name: "Setup", help: []KeyHelp{
			{Keys: []string{"r"}, Description: "refresh setup checks " + long},
			{Keys: []string{"enter"}, Description: "run provider setup fix " + long},
		}},
		&stubScreen{name: "Agents", help: []KeyHelp{{Keys: []string{"n"}, Description: "spawn agent " + long}}},
	)
	updated, _ := shell.Update(tea.WindowSizeMsg{Width: 32, Height: 8})
	shell = updated.(*Shell)
	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	shell = updated.(*Shell)

	view := shell.View()
	lines := strings.Split(view, "\n")
	if len(lines) > 8 {
		t.Fatalf("help overlay height = %d, want <= 8:\n%s", len(lines), view)
	}
	for _, line := range lines {
		if got := lipgloss.Width(line); got > 32 {
			t.Fatalf("help overlay line width %d exceeds 32:\n%q\n\nfull output:\n%s", got, line, view)
		}
	}
	collapsed := strings.Join(strings.Fields(view), " ")
	for _, want := range []string{"Admin help", "Setup", "refresh", "omitted", "help"} {
		if !strings.Contains(collapsed, want) {
			t.Fatalf("help overlay missing %q:\n%s", want, view)
		}
	}
}

// TestAdminShell_QuitKeyExitsCleanly drives both 'q' and Ctrl-C and asserts
// the program finishes within the expected window with no recorded error.
func TestAdminShell_QuitKeyExitsCleanly(t *testing.T) {
	for _, tc := range []struct {
		name string
		key  tea.KeyMsg
	}{
		{name: "q", key: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}},
		{name: "ctrl_c", key: tea.KeyMsg{Type: tea.KeyCtrlC}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			shell := New(&stubScreen{name: "Setup"})
			tm := teatest.NewTestModel(t, shell, teatest.WithInitialTermSize(80, 24))
			tm.Send(tc.key)
			tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
			if err := shell.Err(); err != nil {
				t.Errorf("Err() = %v, want nil", err)
			}
		})
	}
}

func waitForActive(t *testing.T, shell *Shell, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if shell.ActiveIndex() == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("ActiveIndex = %d, want %d", shell.ActiveIndex(), want)
}

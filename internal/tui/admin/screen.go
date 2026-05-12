package admin

import tea "github.com/charmbracelet/bubbletea"

// Screen is one tab in the unified admin TUI. Each Screen is a self-
// contained Bubble Tea sub-model that the shell delegates to while the
// tab is focused. The shell owns navigation, the tab bar, the status
// bar, and global keybindings; screens own their own content + keys.
type Screen interface {
	// Title is the short label rendered in the tab bar.
	Title() string
	// Init returns an initial tea.Cmd run when the screen first becomes
	// focused (or when the shell starts up if the screen is index 0).
	Init() tea.Cmd
	// Update receives messages while the screen is focused. Returns the
	// next Screen state and any follow-up command.
	Update(tea.Msg) (Screen, tea.Cmd)
	// View renders the screen body. The shell wraps this in the tab bar
	// and status bar — screens should not draw chrome themselves.
	View() string
	// ShortHelp returns the keybinding entries this screen contributes to
	// the global help overlay (triggered by '?').
	ShortHelp() []KeyHelp
}

// KeyHelp is one row in the help overlay: a list of keys and a short
// description of what they do on the active screen.
type KeyHelp struct {
	Keys        []string
	Description string
}

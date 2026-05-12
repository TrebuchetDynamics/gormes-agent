package admin

import tea "github.com/charmbracelet/bubbletea"

type welcomeScreen struct{}

// NewWelcomeScreen returns the placeholder screen used by the first admin TUI
// shell slice. Concrete Setup, Chat, and Agents screens land in follow-up rows.
func NewWelcomeScreen() Screen {
	return welcomeScreen{}
}

func (welcomeScreen) Title() string { return "Welcome" }

func (welcomeScreen) Init() tea.Cmd { return nil }

func (welcomeScreen) Update(tea.Msg) (Screen, tea.Cmd) { return welcomeScreen{}, nil }

func (welcomeScreen) View() string {
	return "Gormes admin\n\nUse Tab to move between admin screens as they are enabled."
}

func (welcomeScreen) ShortHelp() []KeyHelp {
	return []KeyHelp{
		{Keys: []string{"tab", "shift+tab"}, Description: "cycle screens"},
		{Keys: []string{"1-9"}, Description: "jump to screen"},
	}
}

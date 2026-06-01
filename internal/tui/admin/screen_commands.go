package admin

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/admin/commands"
	commandcontracts "github.com/TrebuchetDynamics/gormes-agent/internal/tui/admin/contracts/commands"
)

// CommandEntry is one command surfaced in the unified admin TUI command
// catalog. Root aliases preserve the original admin package API while the
// focused contracts package lets command catalog producers avoid depending on
// concrete admin screen implementations.
type CommandEntry = commandcontracts.Entry

// CommandRunner executes one safe, read-only command selected from the command
// catalog. cmd/gormes injects the production runner so this package stays
// independent of Cobra.
type CommandRunner = commandcontracts.Runner

// CommandRunResult is the inline output rendered after a safe command run.
type CommandRunResult = commandcontracts.RunResult

// CommandsScreen lists the Gormes CLI command tree inside the admin TUI.
type CommandsScreen = commands.Screen

// CommandScreenOption configures the command catalog tab.
type CommandScreenOption = commands.Option

// WithCommandRunner lets the command catalog execute safe commands inline.
func WithCommandRunner(runner CommandRunner) CommandScreenOption {
	return commands.WithCommandRunner(runner)
}

// NewCommandsScreen returns the command catalog tab.
func NewCommandsScreen(entries []CommandEntry, opts ...CommandScreenOption) *CommandsScreen {
	return commands.NewScreen(entries, opts...)
}

func cloneCommandEntries(entries []CommandEntry) []CommandEntry {
	return commands.CloneEntries(entries)
}

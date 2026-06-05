package contracts

// CommandEntry is one command surfaced in the unified admin TUI command
// catalog. cmd/gormes builds these from the live Cobra command tree so the
// admin TUI stays independent of Cobra and command implementations.
type CommandEntry struct {
	Path     string
	Use      string
	Short    string
	Runnable bool
	RunLabel string
}

// CommandRunner executes one safe, read-only command selected from the command
// catalog. cmd/gormes injects the production runner so this package stays
// independent of Cobra.
type CommandRunner func(CommandEntry) CommandRunResult

// CommandRunResult is the inline output rendered after a safe command run.
type CommandRunResult struct {
	RunLabel string
	Output   string
	Error    string
	ExitCode int
}

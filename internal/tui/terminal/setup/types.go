package setup

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/setup/fileops"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/setup/truecolor"
)

type TerminalSetupFileOps = fileops.Ops

type TerminalSetupOptions struct {
	Env      map[string]string
	HomeDir  string
	Platform string
	FileOps  TerminalSetupFileOps
}

type TerminalSetupResult struct {
	Success         bool
	RequiresRestart bool
	Message         string
	Evidence        string
	Path            string
}

type TruecolorResult = truecolor.Result

type TerminalParityHint struct {
	Key     string
	Message string
}

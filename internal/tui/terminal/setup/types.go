package setup

import (
	"os"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/setup/truecolor"
)

type TerminalSetupFileOps struct {
	MkdirAll  func(path string, perm os.FileMode) error
	ReadFile  func(path string) ([]byte, error)
	WriteFile func(path string, data []byte, perm os.FileMode) error
	CopyFile  func(src, dst string) error
}

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

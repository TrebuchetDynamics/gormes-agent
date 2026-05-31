package setup

import "os"

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

type TruecolorResult struct {
	Force    bool
	Set      map[string]string
	Unset    []string
	Evidence string
}

type TerminalParityHint struct {
	Key     string
	Message string
}

//go:build !windows

package tools

import (
	"os"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/filesystem"
)

func lockMemoryFile(file *os.File) (func() error, error) {
	return filesystem.LockMemoryFile(file)
}

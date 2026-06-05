package filesystem

import (
	"os"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/filesystem/memorylock"
)

func LockMemoryFile(file *os.File) (func() error, error) {
	return memorylock.LockMemoryFile(file)
}

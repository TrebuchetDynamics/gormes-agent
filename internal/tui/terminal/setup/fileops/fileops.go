package fileops

import (
	"fmt"
	"os"
	"time"
)

// Ops contains the filesystem operations used by terminal setup.
type Ops struct {
	MkdirAll  func(path string, perm os.FileMode) error
	ReadFile  func(path string) ([]byte, error)
	WriteFile func(path string, data []byte, perm os.FileMode) error
	CopyFile  func(src, dst string) error
}

// WithDefaults fills nil operations with os-backed implementations.
func WithDefaults(ops Ops) Ops {
	if ops.MkdirAll == nil {
		ops.MkdirAll = os.MkdirAll
	}
	if ops.ReadFile == nil {
		ops.ReadFile = os.ReadFile
	}
	if ops.WriteFile == nil {
		ops.WriteFile = os.WriteFile
	}
	if ops.CopyFile == nil {
		ops.CopyFile = Copy
	}
	return ops
}

// Copy copies a file body to dst using the terminal setup's default file mode.
func Copy(src, dst string) error {
	body, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, body, 0o644)
}

// BackupPath returns the backup path used before mutating an existing config file.
func BackupPath(path string) string {
	return fmt.Sprintf("%s.gormes-backup-%d", path, time.Now().Unix())
}

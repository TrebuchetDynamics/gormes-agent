package setup

import (
	"fmt"
	"os"
	"time"
)

func (ops TerminalSetupFileOps) withDefaults() TerminalSetupFileOps {
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
		ops.CopyFile = copyFile
	}
	return ops
}

func copyFile(src, dst string) error {
	body, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, body, 0o644)
}

func backupPath(path string) string {
	return fmt.Sprintf("%s.gormes-backup-%d", path, time.Now().Unix())
}

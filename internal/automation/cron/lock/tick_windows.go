//go:build windows

package lock

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	configpaths "github.com/TrebuchetDynamics/gormes-agent/internal/config/paths"
)

type TickLock struct {
	file *os.File
	path string
}

func DefaultPath() string {
	return filepath.Join(configpaths.GormesHome(), "cron", ".tick.lock")
}

func Acquire(path string) (*TickLock, bool, error) {
	rawPath := strings.TrimSpace(path)
	if rawPath == "" {
		return &TickLock{}, true, nil
	}
	path = filepath.Clean(rawPath)
	if path == "." {
		return &TickLock{}, true, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, false, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if _, err := file.WriteString(strconv.Itoa(os.Getpid())); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return nil, false, err
	}
	return &TickLock{file: file, path: path}, true, nil
}

func (l *TickLock) Release() {
	if l == nil {
		return
	}
	if l.file != nil {
		_ = l.file.Close()
	}
	if l.path != "" {
		_ = os.Remove(l.path)
	}
}

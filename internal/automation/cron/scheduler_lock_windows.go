//go:build windows

package cron

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	configpaths "github.com/TrebuchetDynamics/gormes-agent/internal/config/paths"
)

type cronTickLock struct {
	file *os.File
	path string
}

func defaultCronTickLockPath() string {
	return filepath.Join(configpaths.GormesHome(), "cron", ".tick.lock")
}

func acquireCronTickLock(path string) (*cronTickLock, bool, error) {
	rawPath := strings.TrimSpace(path)
	if rawPath == "" {
		return &cronTickLock{}, true, nil
	}
	path = filepath.Clean(rawPath)
	if path == "." {
		return &cronTickLock{}, true, nil
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
	return &cronTickLock{file: file, path: path}, true, nil
}

func (l *cronTickLock) Release() {
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

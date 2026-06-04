//go:build !windows

package cron

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	configpaths "github.com/TrebuchetDynamics/gormes-agent/internal/config/paths"
	"golang.org/x/sys/unix"
)

type cronTickLock struct {
	file *os.File
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
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, false, err
	}
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = file.Close()
		if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if err := file.Truncate(0); err != nil {
		_ = unix.Flock(int(file.Fd()), unix.LOCK_UN)
		_ = file.Close()
		return nil, false, err
	}
	if _, err := file.WriteString(strconv.Itoa(os.Getpid())); err != nil {
		_ = unix.Flock(int(file.Fd()), unix.LOCK_UN)
		_ = file.Close()
		return nil, false, err
	}
	return &cronTickLock{file: file}, true, nil
}

func (l *cronTickLock) Release() {
	if l == nil || l.file == nil {
		return
	}
	_ = unix.Flock(int(l.file.Fd()), unix.LOCK_UN)
	_ = l.file.Close()
}

//go:build !windows

package lock

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	configpaths "github.com/TrebuchetDynamics/gormes-agent/internal/config/paths"
	"golang.org/x/sys/unix"
)

type TickLock struct {
	file *os.File
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
	return &TickLock{file: file}, true, nil
}

func (l *TickLock) Release() {
	if l == nil || l.file == nil {
		return
	}
	_ = unix.Flock(int(l.file.Fd()), unix.LOCK_UN)
	_ = l.file.Close()
}

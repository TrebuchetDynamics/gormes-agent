//go:build !windows

package osproc

import (
	"errors"
	"syscall"
)

// Alive reports whether pid appears to reference a live process.
func Alive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, syscall.Signal(0))
	return err == nil || errors.Is(err, syscall.EPERM)
}

// Terminate sends the platform's normal termination signal to pid.
func Terminate(pid int) error {
	if pid <= 0 {
		return nil
	}
	return syscall.Kill(pid, syscall.SIGTERM)
}

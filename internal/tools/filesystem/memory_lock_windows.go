//go:build windows

package filesystem

import (
	"math"
	"os"

	"golang.org/x/sys/windows"
)

func LockMemoryFile(file *os.File) (func() error, error) {
	handle := windows.Handle(file.Fd())
	overlapped := &windows.Overlapped{}
	if err := windows.LockFileEx(handle, windows.LOCKFILE_EXCLUSIVE_LOCK, 0, math.MaxUint32, math.MaxUint32, overlapped); err != nil {
		return nil, err
	}
	return func() error {
		return windows.UnlockFileEx(handle, 0, math.MaxUint32, math.MaxUint32, overlapped)
	}, nil
}

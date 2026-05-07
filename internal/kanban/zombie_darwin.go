//go:build darwin

package kanban

import "syscall"

func isWorkerProcessZombie(pid int) bool {
	if pid <= 0 {
		return true
	}
	err := syscall.Kill(pid, 0)
	return err != nil
}

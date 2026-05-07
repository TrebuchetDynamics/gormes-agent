//go:build !darwin

package kanban

func isWorkerProcessZombie(pid int) bool {
	return false
}

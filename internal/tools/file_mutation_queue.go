package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/filesystem"

var defaultFileMutationQueue = NewFileMutationQueue()

type FileMutationQueue = filesystem.FileMutationQueue

func NewFileMutationQueue() *FileMutationQueue {
	return filesystem.NewFileMutationQueue()
}

func fileTaskMutationQueue(cfg FileTaskToolConfig) *FileMutationQueue {
	if cfg.MutationQueue != nil {
		return cfg.MutationQueue
	}
	return defaultFileMutationQueue
}

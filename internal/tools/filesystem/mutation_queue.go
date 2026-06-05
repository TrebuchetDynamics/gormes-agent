package filesystem

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/filesystem/mutation"

type FileMutationQueue = mutation.FileMutationQueue

func NewFileMutationQueue() *FileMutationQueue {
	return mutation.NewFileMutationQueue()
}

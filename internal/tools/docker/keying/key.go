package keying

import "strings"

// ContainerRequest describes the task scope used to key a Docker
// execution environment before the live Docker backend exists.
type ContainerRequest struct {
	TaskID     string
	IsSubagent bool
	IsRollout  bool
}

// ContainerKey returns the reusable Docker container key for a request.
func ContainerKey(req ContainerRequest) string {
	taskID := strings.TrimSpace(req.TaskID)
	if taskID != "" {
		return taskID
	}
	if req.IsSubagent || req.IsRollout {
		return ""
	}
	return "default"
}

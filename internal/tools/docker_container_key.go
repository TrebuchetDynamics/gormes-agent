package tools

import "strings"

// DockerContainerRequest describes the task scope used to key a Docker
// execution environment before the live Docker backend exists.
type DockerContainerRequest struct {
	TaskID     string
	IsSubagent bool
	IsRollout  bool
}

// DockerContainerKey returns the reusable Docker container key for a request.
func DockerContainerKey(req DockerContainerRequest) string {
	taskID := strings.TrimSpace(req.TaskID)
	if taskID != "" {
		return taskID
	}
	if req.IsSubagent || req.IsRollout {
		return ""
	}
	return "default"
}

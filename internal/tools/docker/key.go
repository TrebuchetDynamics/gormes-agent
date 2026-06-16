package docker

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/docker/keying"

// DockerContainerRequest describes the task scope used to key a Docker
// execution environment before the live Docker backend exists.
type DockerContainerRequest = keying.ContainerRequest

// DockerContainerKey returns the reusable Docker container key for a request.
func DockerContainerKey(req DockerContainerRequest) string {
	return keying.ContainerKey(req)
}

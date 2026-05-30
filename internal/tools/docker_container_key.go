package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/docker"

type DockerContainerRequest = docker.DockerContainerRequest

func DockerContainerKey(req DockerContainerRequest) string {
	return docker.DockerContainerKey(req)
}

package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/docker"

type DockerExecConfig = docker.DockerExecConfig
type DockerExecResult = docker.DockerExecResult
type DockerRunner = docker.DockerRunner
type DockerExecBackend = docker.DockerExecBackend

func NewDockerExecBackend(runner DockerRunner, config DockerExecConfig, hostPaths []string) *DockerExecBackend {
	return docker.NewDockerExecBackend(runner, config, hostPaths)
}

package docker

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/docker/contract"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/docker/execution"
)

// DockerExecConfig holds configuration for executing commands in a Docker container.
type DockerExecConfig = contract.ExecConfig

// DockerExecResult holds the captured output of a Docker exec command.
type DockerExecResult = contract.ExecResult

// DockerRunner abstracts Docker container execution for testability.
type DockerRunner = contract.Runner

// DockerExecBackend executes commands inside Docker containers with mount
// policy enforcement, image resolution, env filtering, and timeout cleanup.
type DockerExecBackend = execution.Backend

// NewDockerExecBackend creates a backend backed by the given runner.
func NewDockerExecBackend(runner DockerRunner, config DockerExecConfig, hostPaths []string) *DockerExecBackend {
	return execution.NewBackend(runner, config, hostPaths)
}

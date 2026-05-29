package tools

import (
	"context"
	"fmt"
	"time"
)

// DockerExecConfig holds configuration for executing commands in a Docker container.
type DockerExecConfig struct {
	Image              string
	CWD                string
	Timeout            time.Duration
	Env                map[string]string
	EnvAllowlist       []string
	ContainerName      string
	WorkspacePath      string
	ContainerWorkspace string
}

// DockerExecResult holds the captured output of a Docker exec command.
type DockerExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
}

// DockerRunner abstracts Docker container execution for testability.
type DockerRunner interface {
	Run(ctx context.Context, image string, cmd []string, env map[string]string,
		mounts []MountEntry, cwd string, timeout time.Duration) (DockerExecResult, error)
}

// DockerExecBackend executes commands inside Docker containers with mount
// policy enforcement, image resolution, env filtering, and timeout cleanup.
type DockerExecBackend struct {
	runner DockerRunner
	config DockerExecConfig
	policy MountPolicy
}

// NewDockerExecBackend creates a backend backed by the given runner.
func NewDockerExecBackend(runner DockerRunner, config DockerExecConfig, hostPaths []string) *DockerExecBackend {
	return &DockerExecBackend{
		runner: runner,
		config: config,
		policy: DefaultMountPolicy(hostPaths),
	}
}

// resolveImage returns the configured image or a Hermes-compatible default.
func (b *DockerExecBackend) resolveImage() string {
	if b.config.Image != "" {
		return b.config.Image
	}
	return "hermes-sandbox:latest"
}

// filteredEnv returns only allowlisted environment variables.
func (b *DockerExecBackend) filteredEnv() map[string]string {
	if len(b.config.EnvAllowlist) == 0 {
		return b.config.Env
	}
	filtered := make(map[string]string, len(b.config.EnvAllowlist))
	for _, key := range b.config.EnvAllowlist {
		if val, ok := b.config.Env[key]; ok {
			filtered[key] = val
		}
	}
	return filtered
}

// Execute runs a command inside a Docker container with mount policy,
// image resolution, env filtering, and timeout.
func (b *DockerExecBackend) Execute(ctx context.Context, command string, args []string) (DockerExecResult, error) {
	mounts := b.policy.AllowedMounts(b.config.WorkspacePath, b.config.ContainerWorkspace)

	cwd := b.config.CWD
	if cwd == "" {
		cwd = b.config.ContainerWorkspace
		if cwd == "" {
			cwd = "/workspace"
		}
	}

	timeout := b.config.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	image := b.resolveImage()
	env := b.filteredEnv()

	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := append([]string{command}, args...)
	result, err := b.runner.Run(ctxWithTimeout, image, cmd, env, mounts, cwd, timeout)
	if err != nil {
		return result, fmt.Errorf("docker exec: %w", err)
	}

	return result, nil
}

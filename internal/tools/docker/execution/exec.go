package execution

import (
	"context"
	"fmt"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/docker/contract"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/docker/policy"
)

// Backend executes commands inside Docker containers with mount policy
// enforcement, image resolution, env filtering, and timeout cleanup.
type Backend struct {
	runner contract.Runner
	config contract.ExecConfig
	policy policy.MountPolicy
}

// NewBackend creates a backend backed by the given runner.
func NewBackend(runner contract.Runner, config contract.ExecConfig, hostPaths []string) *Backend {
	return &Backend{
		runner: runner,
		config: config,
		policy: policy.DefaultMountPolicy(hostPaths),
	}
}

// ResolveImage returns the configured image or a Hermes-compatible default.
func (b *Backend) ResolveImage() string {
	if b.config.Image != "" {
		return b.config.Image
	}
	return "hermes-sandbox:latest"
}

// FilteredEnv returns only allowlisted environment variables.
func (b *Backend) FilteredEnv() map[string]string {
	return FilterEnvSnapshot(b.config.Env, b.config.EnvAllowlist)
}

func FilterEnvSnapshot(env map[string]string, allowlist []string) map[string]string {
	if len(allowlist) == 0 {
		return copyEnv(env)
	}
	filtered := make(map[string]string, len(allowlist))
	for _, key := range allowlist {
		if val, ok := env[key]; ok {
			filtered[key] = val
		}
	}
	return filtered
}

func copyEnv(env map[string]string) map[string]string {
	if env == nil {
		return nil
	}
	copied := make(map[string]string, len(env))
	for key, val := range env {
		copied[key] = val
	}
	return copied
}

// Execute runs a command inside a Docker container with mount policy,
// image resolution, env filtering, and timeout.
func (b *Backend) Execute(ctx context.Context, command string, args []string) (contract.ExecResult, error) {
	mounts := b.policy.AllowedMounts(b.config.WorkspacePath, b.config.ContainerWorkspace)

	cwd := b.config.CWD
	if cwd == "" {
		cwd = policy.ContainerWorkspacePath(b.config.ContainerWorkspace)
	}

	timeout := b.config.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	image := b.ResolveImage()
	env := b.FilteredEnv()

	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := append([]string{command}, args...)
	result, err := b.runner.Run(ctxWithTimeout, image, cmd, env, mounts, cwd, timeout)
	if err != nil {
		return result, fmt.Errorf("docker exec: %w", err)
	}

	return result, nil
}

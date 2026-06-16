package contract

import (
	"context"
	"time"
)

// ExecConfig holds configuration for executing commands in a Docker container.
type ExecConfig struct {
	Image              string
	CWD                string
	Timeout            time.Duration
	Env                map[string]string
	EnvAllowlist       []string
	ContainerName      string
	WorkspacePath      string
	ContainerWorkspace string
}

// ExecResult holds the captured output of a Docker exec command.
type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
}

// MountEntry describes a resolved mount from host to container.
type MountEntry struct {
	HostPath      string
	ContainerPath string
	ReadOnly      bool
}

// Runner abstracts Docker container execution for testability.
type Runner interface {
	Run(ctx context.Context, image string, cmd []string, env map[string]string,
		mounts []MountEntry, cwd string, timeout time.Duration) (ExecResult, error)
}

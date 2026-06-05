package gitrunner

import (
	"context"
	"errors"
	"os/exec"
)

// Runner executes git commands from an update-managed checkout.
type Runner interface {
	RunGit(ctx context.Context, cwd string, args ...string) Result
}

// Result captures the stdout/stderr/error triple returned by a git command.
type Result struct {
	Stdout string
	Stderr string
	Err    error
}

// RealRunner shells out to the configured git binary.
type RealRunner struct {
	Git string
}

func (r RealRunner) RunGit(ctx context.Context, cwd string, args ...string) Result {
	git := r.Git
	if git == "" {
		git = "git"
	}
	cmd := exec.CommandContext(ctx, git, args...)
	cmd.Dir = cwd
	out, err := cmd.Output()
	result := Result{Stdout: string(out), Err: err}
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			result.Stderr = string(exit.Stderr)
		}
	}
	return result
}

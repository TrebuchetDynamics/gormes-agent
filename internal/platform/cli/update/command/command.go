package command

import (
	"context"
	"os"
	"os/exec"
	"strings"
)

type Runner interface {
	RunCommand(ctx context.Context, cwd string, env []string, name string, args ...string) Result
}

type Result struct {
	Stdout string
	Stderr string
	Err    error
}

type RealRunner struct{}

func (RealRunner) RunCommand(ctx context.Context, cwd string, env []string, name string, args ...string) Result {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = cwd
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	out, err := cmd.CombinedOutput()
	result := Result{Stdout: string(out), Err: err}
	if err != nil {
		result.Stderr = string(out)
	}
	return result
}

func FailureDetail(result Result) string {
	parts := []string{}
	if result.Err != nil {
		parts = append(parts, result.Err.Error())
	}
	if stderr := strings.TrimSpace(result.Stderr); stderr != "" {
		parts = append(parts, stderr)
	} else if stdout := strings.TrimSpace(result.Stdout); stdout != "" {
		parts = append(parts, stdout)
	}
	if len(parts) == 0 {
		return "command failed"
	}
	return strings.Join(parts, ": ")
}

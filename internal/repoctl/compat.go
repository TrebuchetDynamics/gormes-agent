package repoctl

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

type CommandResult struct {
	Stdout string
	Stderr string
}

type Runner interface {
	Run(ctx context.Context, dir string, name string, args ...string) (CommandResult, error)
}

type RunnerFunc func(ctx context.Context, dir string, name string, args ...string) (CommandResult, error)

func (f RunnerFunc) Run(ctx context.Context, dir string, name string, args ...string) (CommandResult, error) {
	return f(ctx, dir, name, args...)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, dir string, name string, args ...string) (CommandResult, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return CommandResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}, err
}

type Go122Options struct {
	Root   string
	Runner Runner
	Stdout io.Writer
}

func CheckGo122(ctx context.Context, opts Go122Options) error {
	if opts.Root == "" {
		return errors.New("root path is required")
	}
	runner := opts.Runner
	if runner == nil {
		runner = ExecRunner{}
	}
	stdout := opts.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}

	if _, err := runner.Run(ctx, opts.Root, "docker", "--version"); err == nil {
		result, err := runner.Run(ctx, opts.Root, "docker", "run", "--rm", "-v", opts.Root+":/src", "-w", "/src", "golang:1.22-alpine", "go", "build", "./cmd/gormes")
		if err != nil {
			writeGo122Failure(stdout, "Docker golang:1.22-alpine", result)
			return fmt.Errorf("go 1.22 docker build failed: %w", err)
		}
		writeGo122Success(stdout)
		return nil
	}

	if result, err := runner.Run(ctx, opts.Root, "go", "install", "golang.org/dl/go1.22.10@latest"); err != nil {
		writeGo122Failure(stdout, "downloaded Go 1.22 toolchain install", result)
		return fmt.Errorf("go1.22.10 install failed: %w", err)
	}
	if result, err := runner.Run(ctx, opts.Root, "go1.22.10", "download"); err != nil {
		writeGo122Failure(stdout, "downloaded Go 1.22 toolchain download", result)
		return fmt.Errorf("go1.22.10 download failed: %w", err)
	}
	result, err := runner.Run(ctx, opts.Root, "go1.22.10", "build", "./cmd/gormes")
	if err != nil {
		writeGo122Failure(stdout, "downloaded Go 1.22 toolchain", result)
		return fmt.Errorf("go1.22.10 build failed: %w", err)
	}
	writeGo122Success(stdout)
	return nil
}

func writeGo122Success(w io.Writer) {
	fmt.Fprintln(w, "=== Decision data for 'Portability vs. Progress' ===")
	fmt.Fprintln(w, "  Go 1.22 builds cleanly; no action needed")
}

func writeGo122Failure(w io.Writer, path string, result CommandResult) {
	fmt.Fprintln(w, "=== Decision data for 'Portability vs. Progress' ===")
	fmt.Fprintf(w, "  Go 1.22 build failed via %s\n", path)
	fmt.Fprintf(w, "  stdout:\n%s\n", result.Stdout)
	fmt.Fprintf(w, "  stderr:\n%s\n", result.Stderr)
}

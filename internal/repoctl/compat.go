package repoctl

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
			writeGo122Failure(stdout, "build", "Docker golang:1.22-alpine", result)
			return fmt.Errorf("go 1.22 docker build failed: %w", err)
		}
		writeGo122Success(stdout)
		return nil
	}

	if result, err := runner.Run(ctx, opts.Root, "go", "install", "golang.org/dl/go1.22.10@latest"); err != nil {
		writeGo122Failure(stdout, "install", "downloaded Go 1.22 toolchain", result)
		return fmt.Errorf("go1.22.10 install failed: %w", err)
	}
	wrapper := "go1.22.10"
	if result, err := runGo122Tool(ctx, opts.Root, runner, &wrapper, "download"); err != nil {
		writeGo122Failure(stdout, "download", "downloaded Go 1.22 toolchain", result)
		return fmt.Errorf("go1.22.10 download failed: %w", err)
	}
	result, err := runGo122Tool(ctx, opts.Root, runner, &wrapper, "build", "./cmd/gormes")
	if err != nil {
		writeGo122Failure(stdout, "build", "downloaded Go 1.22 toolchain", result)
		return fmt.Errorf("go1.22.10 build failed: %w", err)
	}
	writeGo122Success(stdout)
	return nil
}

func runGo122Tool(ctx context.Context, root string, runner Runner, wrapper *string, args ...string) (CommandResult, error) {
	result, err := runner.Run(ctx, root, *wrapper, args...)
	if err == nil || !isCommandNotFound(err) {
		return result, err
	}

	resolved, resolveErr := resolveInstalledGo122Wrapper(ctx, root, runner)
	if resolveErr != nil {
		return result, err
	}
	*wrapper = resolved
	return runner.Run(ctx, root, *wrapper, args...)
}

func resolveInstalledGo122Wrapper(ctx context.Context, root string, runner Runner) (string, error) {
	result, err := runner.Run(ctx, root, "go", "env", "GOBIN")
	if err != nil {
		return "", err
	}
	if gobin := strings.TrimSpace(result.Stdout); gobin != "" {
		return filepath.Join(gobin, "go1.22.10"), nil
	}

	result, err = runner.Run(ctx, root, "go", "env", "GOPATH")
	if err != nil {
		return "", err
	}
	gopath := strings.TrimSpace(result.Stdout)
	if gopath == "" {
		return "", errors.New("go env GOPATH returned empty path")
	}
	return filepath.Join(gopath, "bin", "go1.22.10"), nil
}

func isCommandNotFound(err error) bool {
	return errors.Is(err, exec.ErrNotFound) || os.IsNotExist(err)
}

func writeGo122Success(w io.Writer) {
	fmt.Fprintln(w, "=== Decision data for 'Portability vs. Progress' ===")
	fmt.Fprintln(w, "  Go 1.22 builds cleanly; no action needed")
}

func writeGo122Failure(w io.Writer, phase string, path string, result CommandResult) {
	fmt.Fprintln(w, "=== Decision data for 'Portability vs. Progress' ===")
	fmt.Fprintf(w, "  Go 1.22 %s failed via %s\n", phase, path)
	fmt.Fprintf(w, "  stdout:\n%s\n", result.Stdout)
	fmt.Fprintf(w, "  stderr:\n%s\n", result.Stderr)
}

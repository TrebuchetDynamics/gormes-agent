package docker

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type fakeDockerRunner struct {
	lastImage   string
	lastCmd     []string
	lastEnv     map[string]string
	lastMounts  []MountEntry
	lastCWD     string
	lastTimeout time.Duration

	result DockerExecResult
	err    error
}

func (f *fakeDockerRunner) Run(ctx context.Context, image string, cmd []string,
	env map[string]string, mounts []MountEntry, cwd string,
	timeout time.Duration) (DockerExecResult, error) {
	f.lastImage = image
	f.lastCmd = cmd
	f.lastEnv = env
	f.lastMounts = mounts
	f.lastCWD = cwd
	f.lastTimeout = timeout

	if f.err != nil {
		return DockerExecResult{}, f.err
	}
	return f.result, nil
}

func TestDockerExec_ImageResolution(t *testing.T) {
	t.Run("uses configured image", func(t *testing.T) {
		runner := &fakeDockerRunner{
			result: DockerExecResult{Stdout: "ok", ExitCode: 0},
		}
		backend := NewDockerExecBackend(runner, DockerExecConfig{
			Image: "ubuntu:22.04",
		}, nil)
		_, err := backend.Execute(context.Background(), "echo", []string{"hello"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if runner.lastImage != "ubuntu:22.04" {
			t.Errorf("expected image ubuntu:22.04, got %s", runner.lastImage)
		}
	})

	t.Run("uses default image when config image is empty", func(t *testing.T) {
		runner := &fakeDockerRunner{
			result: DockerExecResult{Stdout: "ok", ExitCode: 0},
		}
		backend := NewDockerExecBackend(runner, DockerExecConfig{}, nil)
		_, err := backend.Execute(context.Background(), "echo", []string{"hello"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if runner.lastImage != "hermes-sandbox:latest" {
			t.Errorf("expected default image hermes-sandbox:latest, got %s", runner.lastImage)
		}
	})
}

func TestDockerExec_MountPolicyAllowlist(t *testing.T) {
	t.Run("allows configured host paths", func(t *testing.T) {
		runner := &fakeDockerRunner{
			result: DockerExecResult{Stdout: "ok", ExitCode: 0},
		}
		hostPaths := []string{"/tmp/data", "/home/user"}
		backend := NewDockerExecBackend(runner, DockerExecConfig{
			WorkspacePath:      "/tmp/data",
			ContainerWorkspace: "/workspace",
		}, hostPaths)

		_, err := backend.Execute(context.Background(), "ls", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(runner.lastMounts) != 2 {
			t.Fatalf("expected 2 mounts, got %d", len(runner.lastMounts))
		}
	})

	t.Run("deduplicates cleaned host paths before mounting", func(t *testing.T) {
		runner := &fakeDockerRunner{
			result: DockerExecResult{Stdout: "ok", ExitCode: 0},
		}
		backend := NewDockerExecBackend(runner, DockerExecConfig{
			WorkspacePath:      "/tmp/data",
			ContainerWorkspace: "/workspace",
		}, []string{"/tmp/data", "/tmp/data/", "/tmp/../tmp/data", "/tmp/other"})

		_, err := backend.Execute(context.Background(), "ls", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(runner.lastMounts) != 2 {
			t.Fatalf("expected 2 unique mounts, got %d: %#v", len(runner.lastMounts), runner.lastMounts)
		}
		if got := runner.lastMounts[0]; got.HostPath != "/tmp/data" || got.ContainerPath != "/workspace" || got.ReadOnly {
			t.Fatalf("workspace mount = %#v, want host /tmp/data at /workspace read-write", got)
		}
		if got := runner.lastMounts[1]; got.HostPath != "/tmp/other" || !got.ReadOnly {
			t.Fatalf("second mount = %#v, want read-only /tmp/other", got)
		}
	})

	t.Run("workspace path is read-write", func(t *testing.T) {
		runner := &fakeDockerRunner{
			result: DockerExecResult{Stdout: "ok", ExitCode: 0},
		}
		hostPaths := []string{"/tmp/data"}
		backend := NewDockerExecBackend(runner, DockerExecConfig{
			WorkspacePath:      "/tmp/data",
			ContainerWorkspace: "/workspace",
		}, hostPaths)

		_, err := backend.Execute(context.Background(), "ls", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(runner.lastMounts) != 1 {
			t.Fatalf("expected 1 mount, got %d", len(runner.lastMounts))
		}
		if runner.lastMounts[0].ReadOnly {
			t.Error("expected workspace mount to be read-write")
		}
	})

	t.Run("workspace mount uses default container workspace when omitted", func(t *testing.T) {
		runner := &fakeDockerRunner{
			result: DockerExecResult{Stdout: "ok", ExitCode: 0},
		}
		backend := NewDockerExecBackend(runner, DockerExecConfig{
			WorkspacePath: "/tmp/data",
		}, []string{"/tmp/data"})

		_, err := backend.Execute(context.Background(), "ls", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(runner.lastMounts) != 1 {
			t.Fatalf("expected 1 mount, got %d", len(runner.lastMounts))
		}
		if got := runner.lastMounts[0].ContainerPath; got != "/workspace" {
			t.Fatalf("workspace mount container path = %q, want /workspace", got)
		}
		if runner.lastCWD != "/workspace" {
			t.Fatalf("cwd = %q, want /workspace", runner.lastCWD)
		}
	})

	t.Run("non-workspace paths are read-only", func(t *testing.T) {
		runner := &fakeDockerRunner{
			result: DockerExecResult{Stdout: "ok", ExitCode: 0},
		}
		hostPaths := []string{"/tmp/data", "/home/user"}
		backend := NewDockerExecBackend(runner, DockerExecConfig{
			WorkspacePath:      "/tmp/data",
			ContainerWorkspace: "/workspace",
		}, hostPaths)

		_, err := backend.Execute(context.Background(), "ls", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(runner.lastMounts) != 2 {
			t.Fatalf("expected 2 mounts, got %d", len(runner.lastMounts))
		}
		mountByHost := map[string]MountEntry{}
		for _, m := range runner.lastMounts {
			mountByHost[m.HostPath] = m
		}
		if m, ok := mountByHost["/home/user"]; !ok {
			t.Error("expected /home/user mount")
		} else if !m.ReadOnly {
			t.Error("expected /home/user mount to be read-only")
		}
	})

	t.Run("blocks dangerous system paths", func(t *testing.T) {
		runner := &fakeDockerRunner{
			result: DockerExecResult{Stdout: "ok", ExitCode: 0},
		}
		hostPaths := []string{"/etc/passwd", "/proc/cpuinfo", "/sys/kernel", "/var/run/docker.sock", "/var/run/docker.sock/child"}
		backend := NewDockerExecBackend(runner, DockerExecConfig{}, hostPaths)

		_, err := backend.Execute(context.Background(), "ls", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(runner.lastMounts) != 0 {
			t.Errorf("expected 0 mounts (all blocked), got %d: %v", len(runner.lastMounts), runner.lastMounts)
		}
	})

	t.Run("does not block sibling paths with dangerous prefix names", func(t *testing.T) {
		runner := &fakeDockerRunner{
			result: DockerExecResult{Stdout: "ok", ExitCode: 0},
		}
		hostPaths := []string{"/var/run/docker.sock.backup", "/etcetera/project"}
		backend := NewDockerExecBackend(runner, DockerExecConfig{}, hostPaths)

		_, err := backend.Execute(context.Background(), "ls", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(runner.lastMounts) != 2 {
			t.Fatalf("expected 2 sibling mounts to remain allowed, got %d: %#v", len(runner.lastMounts), runner.lastMounts)
		}
	})
}

func TestDockerExec_EnvPassthrough(t *testing.T) {
	t.Run("passes only allowlisted env vars", func(t *testing.T) {
		runner := &fakeDockerRunner{
			result: DockerExecResult{Stdout: "ok", ExitCode: 0},
		}
		backend := NewDockerExecBackend(runner, DockerExecConfig{
			Env: map[string]string{
				"API_KEY": "secret123",
				"DEBUG":   "1",
				"PATH":    "/usr/bin",
			},
			EnvAllowlist: []string{"DEBUG", "PATH"},
		}, nil)

		_, err := backend.Execute(context.Background(), "env", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if runner.lastEnv["API_KEY"] != "" {
			t.Error("expected API_KEY to be filtered out")
		}
		if runner.lastEnv["DEBUG"] != "1" {
			t.Error("expected DEBUG to be passed through")
		}
		if runner.lastEnv["PATH"] != "/usr/bin" {
			t.Error("expected PATH to be passed through")
		}
		if len(runner.lastEnv) != 2 {
			t.Errorf("expected 2 env vars, got %d", len(runner.lastEnv))
		}
	})

	t.Run("passes all env vars when no allowlist", func(t *testing.T) {
		runner := &fakeDockerRunner{
			result: DockerExecResult{Stdout: "ok", ExitCode: 0},
		}
		backend := NewDockerExecBackend(runner, DockerExecConfig{
			Env: map[string]string{
				"API_KEY": "secret123",
				"DEBUG":   "1",
			},
		}, nil)

		_, err := backend.Execute(context.Background(), "env", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(runner.lastEnv) != 2 {
			t.Errorf("expected 2 env vars, got %d", len(runner.lastEnv))
		}
	})
}

func TestDockerExec_TimeoutCleanup(t *testing.T) {
	t.Run("propagates context deadline error", func(t *testing.T) {
		runner := &fakeDockerRunner{
			err: context.DeadlineExceeded,
		}
		backend := NewDockerExecBackend(runner, DockerExecConfig{
			Timeout: 1 * time.Millisecond,
		}, nil)

		_, err := backend.Execute(context.Background(), "sleep", []string{"10"})
		if err == nil {
			t.Fatal("expected timeout error")
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("expected DeadlineExceeded, got %v", err)
		}
	})

	t.Run("passes timeout duration to runner", func(t *testing.T) {
		runner := &fakeDockerRunner{
			result: DockerExecResult{Stdout: "ok", ExitCode: 0},
		}
		backend := NewDockerExecBackend(runner, DockerExecConfig{
			Timeout: 30 * time.Second,
		}, nil)

		_, err := backend.Execute(context.Background(), "echo", []string{"hi"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if runner.lastTimeout != 30*time.Second {
			t.Errorf("expected 30s timeout, got %v", runner.lastTimeout)
		}
	})

	t.Run("uses default 60s timeout when not configured", func(t *testing.T) {
		runner := &fakeDockerRunner{
			result: DockerExecResult{Stdout: "ok", ExitCode: 0},
		}
		backend := NewDockerExecBackend(runner, DockerExecConfig{}, nil)

		_, err := backend.Execute(context.Background(), "echo", []string{"hi"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if runner.lastTimeout != 60*time.Second {
			t.Errorf("expected default 60s timeout, got %v", runner.lastTimeout)
		}
	})
}

func TestDockerExec_StdoutStderrCapture(t *testing.T) {
	t.Run("captures stdout", func(t *testing.T) {
		runner := &fakeDockerRunner{
			result: DockerExecResult{
				Stdout:   "hello world\n",
				Stderr:   "",
				ExitCode: 0,
			},
		}
		backend := NewDockerExecBackend(runner, DockerExecConfig{}, nil)
		result, err := backend.Execute(context.Background(), "echo", []string{"hello world"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result.Stdout, "hello world") {
			t.Errorf("expected stdout to contain 'hello world', got %q", result.Stdout)
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", result.ExitCode)
		}
	})

	t.Run("captures stderr", func(t *testing.T) {
		errMsg := "command not found"
		runner := &fakeDockerRunner{
			result: DockerExecResult{
				Stdout:   "",
				Stderr:   errMsg,
				ExitCode: 127,
			},
		}
		backend := NewDockerExecBackend(runner, DockerExecConfig{}, nil)
		result, err := backend.Execute(context.Background(), "nonexistent", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Stderr != errMsg {
			t.Errorf("expected stderr %q, got %q", errMsg, result.Stderr)
		}
		if result.ExitCode != 127 {
			t.Errorf("expected exit code 127, got %d", result.ExitCode)
		}
	})

	t.Run("returns runner error", func(t *testing.T) {
		runner := &fakeDockerRunner{
			err: errors.New("container not found"),
		}
		backend := NewDockerExecBackend(runner, DockerExecConfig{}, nil)
		_, err := backend.Execute(context.Background(), "echo", []string{"hi"})
		if err == nil {
			t.Fatal("expected error from runner")
		}
		if !strings.Contains(err.Error(), "container not found") {
			t.Errorf("expected 'container not found' in error, got %v", err)
		}
	})
}

package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestProjectModeCWDResolver(t *testing.T) {
	t.Run("terminal cwd wins when it exists", func(t *testing.T) {
		got := ResolveProjectModeCWD(ProjectModeCWDInput{
			TerminalCWD: "/workspace/project",
			ProcessCWD:  "/workspace/process",
			StagingDir:  "/tmp/staging",
			IsDir:       dirs("/workspace/project", "/workspace/process", "/tmp/staging"),
		})
		if got != "/workspace/project" {
			t.Fatalf("cwd = %q, want terminal cwd", got)
		}
	})

	t.Run("invalid terminal cwd falls back to process cwd", func(t *testing.T) {
		got := ResolveProjectModeCWD(ProjectModeCWDInput{
			TerminalCWD: "/missing/project",
			ProcessCWD:  "/workspace/process",
			StagingDir:  "/tmp/staging",
			IsDir:       dirs("/workspace/process", "/tmp/staging"),
		})
		if got != "/workspace/process" {
			t.Fatalf("cwd = %q, want process cwd", got)
		}
	})

	t.Run("deleted process cwd falls back to staging", func(t *testing.T) {
		got := ResolveProjectModeCWD(ProjectModeCWDInput{
			TerminalCWD: "",
			ProcessCWD:  "/deleted/process",
			StagingDir:  "/tmp/staging",
			IsDir:       dirs("/tmp/staging"),
		})
		if got != "/tmp/staging" {
			t.Fatalf("cwd = %q, want staging dir", got)
		}
	})
}

func TestProjectModePythonResolver(t *testing.T) {
	const fallback = "/usr/bin/python3"

	t.Run("virtualenv wins over conda", func(t *testing.T) {
		got := ResolveProjectModePython(ProjectModePythonInput{
			Env: map[string]string{
				"VIRTUAL_ENV":  "/venv",
				"CONDA_PREFIX": "/conda",
			},
			SystemPython:    fallback,
			IsExecutable:    executable("/venv/bin/python", "/conda/bin/python"),
			IsUsablePython:  usable("/venv/bin/python", "/conda/bin/python"),
			WindowsPlatform: false,
		})
		if got != "/venv/bin/python" {
			t.Fatalf("python = %q, want virtualenv python", got)
		}
	})

	t.Run("conda is used when virtualenv is absent", func(t *testing.T) {
		got := ResolveProjectModePython(ProjectModePythonInput{
			Env: map[string]string{
				"CONDA_PREFIX": "/conda",
			},
			SystemPython:    fallback,
			IsExecutable:    executable("/conda/bin/python"),
			IsUsablePython:  usable("/conda/bin/python"),
			WindowsPlatform: false,
		})
		if got != "/conda/bin/python" {
			t.Fatalf("python = %q, want conda python", got)
		}
	})

	t.Run("missing candidates fall back to system python", func(t *testing.T) {
		got := ResolveProjectModePython(ProjectModePythonInput{
			Env: map[string]string{
				"VIRTUAL_ENV":  "/venv",
				"CONDA_PREFIX": "/conda",
			},
			SystemPython:    fallback,
			IsExecutable:    executable(),
			IsUsablePython:  usable(),
			WindowsPlatform: false,
		})
		if got != fallback {
			t.Fatalf("python = %q, want fallback %q", got, fallback)
		}
	})

	t.Run("unusable virtualenv candidate falls back without checking conda", func(t *testing.T) {
		got := ResolveProjectModePython(ProjectModePythonInput{
			Env: map[string]string{
				"VIRTUAL_ENV":  "/venv",
				"CONDA_PREFIX": "/conda",
			},
			SystemPython:    fallback,
			IsExecutable:    executable("/venv/bin/python", "/conda/bin/python"),
			IsUsablePython:  usable("/conda/bin/python"),
			WindowsPlatform: false,
		})
		if got != fallback {
			t.Fatalf("python = %q, want fallback after unusable virtualenv candidate", got)
		}
	})
}

func TestProjectModeSandbox_Contract(t *testing.T) {
	projectDir := t.TempDir()
	sandbox := newProjectModeSandboxWithHooks(projectModeSandboxHooks{
		lookupEnv: func(key string) (string, bool) {
			if key == "TERMINAL_CWD" {
				return projectDir, true
			}
			return "", false
		},
		getwd: func() (string, error) {
			return "/fallback/process", nil
		},
		isDir: func(path string) bool {
			return path == projectDir || strings.Contains(path, "gormes-execute-code-")
		},
		lookPath: func(file string) (string, error) {
			return "/bin/sh", nil
		},
	})

	result, err := sandbox.Execute(context.Background(), CodeExecutionRequest{
		Language: "sh",
		Code:     `printf '%s' "$PWD"`,
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Status != "success" {
		t.Fatalf("Status = %q, want success; error=%q", result.Status, result.Error)
	}
	if strings.TrimSpace(result.Stdout) != projectDir {
		t.Fatalf("stdout cwd = %q, want project cwd %q", strings.TrimSpace(result.Stdout), projectDir)
	}

	blocked, err := sandbox.Execute(context.Background(), CodeExecutionRequest{
		Language: "sh",
		Code:     "ls /tmp",
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Execute blocked command: %v", err)
	}
	if blocked.Status != "blocked" {
		t.Fatalf("blocked Status = %q, want blocked", blocked.Status)
	}
	if blocked.FilesystemAccess || blocked.NetworkAccess {
		t.Fatalf("blocked result access flags = fs:%v net:%v, want both false", blocked.FilesystemAccess, blocked.NetworkAccess)
	}
	raw, err := json.Marshal(blocked)
	if err != nil {
		t.Fatalf("json.Marshal blocked result: %v", err)
	}
	var envelope map[string]interface{}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatalf("json.Unmarshal blocked result: %v", err)
	}
	for _, field := range []string{"status", "error", "filesystem_access", "network_access"} {
		if _, ok := envelope[field]; !ok {
			t.Fatalf("blocked result missing %q field: %s", field, raw)
		}
	}
}

func dirs(paths ...string) func(string) bool {
	allowed := map[string]bool{}
	for _, path := range paths {
		allowed[path] = true
	}
	return func(path string) bool {
		return allowed[path]
	}
}

func executable(paths ...string) func(string) bool {
	allowed := map[string]bool{}
	for _, path := range paths {
		allowed[path] = true
	}
	return func(path string) bool {
		return allowed[path]
	}
}

func usable(paths ...string) func(string) bool {
	allowed := map[string]bool{}
	for _, path := range paths {
		allowed[path] = true
	}
	return func(path string) bool {
		return allowed[path]
	}
}

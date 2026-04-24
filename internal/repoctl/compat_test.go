package repoctl

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"strings"
	"testing"
)

func TestGo122CompatUsesDockerWhenAvailable(t *testing.T) {
	var calls []runnerCall
	runner := RunnerFunc(func(ctx context.Context, dir string, name string, args ...string) (CommandResult, error) {
		calls = append(calls, runnerCall{dir: dir, name: name, args: append([]string(nil), args...)})
		return CommandResult{}, nil
	})
	var stdout strings.Builder

	err := CheckGo122(context.Background(), Go122Options{
		Root:   "/repo",
		Runner: runner,
		Stdout: &stdout,
	})
	if err != nil {
		t.Fatalf("CheckGo122: %v", err)
	}

	assertCalls(t, calls, []runnerCall{
		{dir: "/repo", name: "docker", args: []string{"--version"}},
		{dir: "/repo", name: "docker", args: []string{"run", "--rm", "-v", "/repo:/src", "-w", "/src", "golang:1.22-alpine", "go", "build", "./cmd/gormes"}},
	})
	if !strings.Contains(stdout.String(), "Go 1.22 builds cleanly") {
		t.Fatalf("stdout missing success data:\n%s", stdout.String())
	}
}

func TestGo122CompatFallsBackToDownloadedToolchain(t *testing.T) {
	var calls []runnerCall
	runner := RunnerFunc(func(ctx context.Context, dir string, name string, args ...string) (CommandResult, error) {
		calls = append(calls, runnerCall{dir: dir, name: name, args: append([]string(nil), args...)})
		if name == "docker" && len(args) == 1 && args[0] == "--version" {
			return CommandResult{Stderr: "docker missing"}, errors.New("docker unavailable")
		}
		return CommandResult{}, nil
	})

	err := CheckGo122(context.Background(), Go122Options{
		Root:   "/repo",
		Runner: runner,
		Stdout: io.Discard,
	})
	if err != nil {
		t.Fatalf("CheckGo122: %v", err)
	}

	assertCalls(t, calls, []runnerCall{
		{dir: "/repo", name: "docker", args: []string{"--version"}},
		{dir: "/repo", name: "go", args: []string{"install", "golang.org/dl/go1.22.10@latest"}},
		{dir: "/repo", name: "go1.22.10", args: []string{"download"}},
		{dir: "/repo", name: "go1.22.10", args: []string{"build", "./cmd/gormes"}},
	})
}

func TestGo122CompatRetriesWrapperFromGoPathWhenBareWrapperIsMissing(t *testing.T) {
	var calls []runnerCall
	runner := RunnerFunc(func(ctx context.Context, dir string, name string, args ...string) (CommandResult, error) {
		calls = append(calls, runnerCall{dir: dir, name: name, args: append([]string(nil), args...)})
		switch {
		case name == "docker":
			return CommandResult{}, errors.New("docker unavailable")
		case name == "go1.22.10":
			return CommandResult{Stderr: "go1.22.10 missing"}, exec.ErrNotFound
		case name == "go" && len(args) == 2 && args[0] == "env" && args[1] == "GOBIN":
			return CommandResult{Stdout: "\n"}, nil
		case name == "go" && len(args) == 2 && args[0] == "env" && args[1] == "GOPATH":
			return CommandResult{Stdout: "/home/test/go\n"}, nil
		}
		return CommandResult{}, nil
	})

	err := CheckGo122(context.Background(), Go122Options{
		Root:   "/repo",
		Runner: runner,
		Stdout: io.Discard,
	})
	if err != nil {
		t.Fatalf("CheckGo122: %v", err)
	}

	assertCalls(t, calls, []runnerCall{
		{dir: "/repo", name: "docker", args: []string{"--version"}},
		{dir: "/repo", name: "go", args: []string{"install", "golang.org/dl/go1.22.10@latest"}},
		{dir: "/repo", name: "go1.22.10", args: []string{"download"}},
		{dir: "/repo", name: "go", args: []string{"env", "GOBIN"}},
		{dir: "/repo", name: "go", args: []string{"env", "GOPATH"}},
		{dir: "/repo", name: "/home/test/go/bin/go1.22.10", args: []string{"download"}},
		{dir: "/repo", name: "/home/test/go/bin/go1.22.10", args: []string{"build", "./cmd/gormes"}},
	})
}

func TestGo122CompatRequiresRoot(t *testing.T) {
	err := CheckGo122(context.Background(), Go122Options{
		Runner: RunnerFunc(func(ctx context.Context, dir string, name string, args ...string) (CommandResult, error) {
			t.Fatal("runner should not be called")
			return CommandResult{}, nil
		}),
		Stdout: io.Discard,
	})
	if err == nil {
		t.Fatal("CheckGo122 returned nil for empty root")
	}
}

func TestGo122CompatReportsDockerBuildFailureDecisionData(t *testing.T) {
	runner := RunnerFunc(func(ctx context.Context, dir string, name string, args ...string) (CommandResult, error) {
		if name == "docker" && len(args) == 1 && args[0] == "--version" {
			return CommandResult{}, nil
		}
		return CommandResult{Stdout: "build stdout", Stderr: "build stderr"}, errors.New("build failed")
	})
	var stdout strings.Builder

	err := CheckGo122(context.Background(), Go122Options{
		Root:   "/repo",
		Runner: runner,
		Stdout: &stdout,
	})
	if err == nil {
		t.Fatal("CheckGo122 returned nil for failed build")
	}
	got := stdout.String()
	for _, want := range []string{"Docker golang:1.22-alpine", "build stdout", "build stderr"} {
		if !strings.Contains(got, want) {
			t.Fatalf("stdout missing %q:\n%s", want, got)
		}
	}
}

func TestGo122CompatReportsFallbackInstallFailureDecisionData(t *testing.T) {
	runner := RunnerFunc(func(ctx context.Context, dir string, name string, args ...string) (CommandResult, error) {
		if name == "docker" {
			return CommandResult{}, errors.New("docker unavailable")
		}
		return CommandResult{Stdout: "install stdout", Stderr: "install stderr"}, errors.New("install failed")
	})
	var stdout strings.Builder

	err := CheckGo122(context.Background(), Go122Options{
		Root:   "/repo",
		Runner: runner,
		Stdout: &stdout,
	})
	if err == nil {
		t.Fatal("CheckGo122 returned nil for failed install")
	}
	assertDecisionDataContains(t, stdout.String(), "Go 1.22 install failed", "install stdout", "install stderr")
	assertDecisionDataDoesNotContain(t, stdout.String(), "Go 1.22 build failed")
}

func TestGo122CompatReportsFallbackDownloadFailureDecisionData(t *testing.T) {
	runner := RunnerFunc(func(ctx context.Context, dir string, name string, args ...string) (CommandResult, error) {
		if name == "docker" {
			return CommandResult{}, errors.New("docker unavailable")
		}
		if name == "go1.22.10" && len(args) == 1 && args[0] == "download" {
			return CommandResult{Stdout: "download stdout", Stderr: "download stderr"}, errors.New("download failed")
		}
		return CommandResult{}, nil
	})
	var stdout strings.Builder

	err := CheckGo122(context.Background(), Go122Options{
		Root:   "/repo",
		Runner: runner,
		Stdout: &stdout,
	})
	if err == nil {
		t.Fatal("CheckGo122 returned nil for failed download")
	}
	assertDecisionDataContains(t, stdout.String(), "Go 1.22 download failed", "download stdout", "download stderr")
	assertDecisionDataDoesNotContain(t, stdout.String(), "Go 1.22 build failed")
}

func TestGo122CompatReportsFallbackBuildFailureDecisionData(t *testing.T) {
	runner := RunnerFunc(func(ctx context.Context, dir string, name string, args ...string) (CommandResult, error) {
		if name == "docker" {
			return CommandResult{}, errors.New("docker unavailable")
		}
		if name == "go1.22.10" && len(args) == 2 && args[0] == "build" {
			return CommandResult{Stdout: "build stdout", Stderr: "build stderr"}, errors.New("build failed")
		}
		return CommandResult{}, nil
	})
	var stdout strings.Builder

	err := CheckGo122(context.Background(), Go122Options{
		Root:   "/repo",
		Runner: runner,
		Stdout: &stdout,
	})
	if err == nil {
		t.Fatal("CheckGo122 returned nil for failed build")
	}
	assertDecisionDataContains(t, stdout.String(), "Go 1.22 build failed", "build stdout", "build stderr")
}

type runnerCall struct {
	dir  string
	name string
	args []string
}

func assertCalls(t *testing.T, got []runnerCall, want []runnerCall) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("calls = %+v, want %+v", got, want)
	}
	for i := range got {
		if got[i].dir != want[i].dir || got[i].name != want[i].name || strings.Join(got[i].args, "\x00") != strings.Join(want[i].args, "\x00") {
			t.Fatalf("call %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func assertDecisionDataContains(t *testing.T, got string, want ...string) {
	t.Helper()
	for _, part := range want {
		if !strings.Contains(got, part) {
			t.Fatalf("stdout missing %q:\n%s", part, got)
		}
	}
}

func assertDecisionDataDoesNotContain(t *testing.T, got string, unwanted string) {
	t.Helper()
	if strings.Contains(got, unwanted) {
		t.Fatalf("stdout unexpectedly contains %q:\n%s", unwanted, got)
	}
}

package repoctl

import (
	"context"
	"errors"
	"io"
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

func TestGo122CompatReportsBuildFailureDecisionData(t *testing.T) {
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

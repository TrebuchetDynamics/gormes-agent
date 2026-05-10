package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCompletionCommand_ZshUsesValidArgumentsExclusion(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())

	stdout, stderr, err := executeRootCommandForTest(newCompletionTestRoot(t), "completion", "zsh")
	if err != nil {
		t.Fatalf("completion zsh error = %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "#compdef gormes") {
		t.Fatalf("zsh completion missing compdef header:\n%s", stdout)
	}
	if !strings.Contains(stdout, "__complete") {
		t.Fatalf("zsh completion missing dynamic completion hook:\n%s", stdout)
	}
	for _, bad := range []string{"(-h --help)", "(-V --version)", "(-p --profile)"} {
		if strings.Contains(stdout, bad) {
			t.Fatalf("zsh completion contains invalid _arguments exclusion group %q:\n%s", bad, stdout)
		}
	}
}

func TestCompletionCommand_GeneratesSupportedShellsWithoutRuntime(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())

	cases := []struct {
		shell string
		want  string
	}{
		{"bash", "gormes"},
		{"fish", "complete -c gormes"},
		{"powershell", "Register-ArgumentCompleter"},
	}
	for _, tc := range cases {
		t.Run(tc.shell, func(t *testing.T) {
			stdout, stderr, err := executeRootCommandForTest(newCompletionTestRoot(t), "completion", tc.shell)
			if err != nil {
				t.Fatalf("completion %s error = %v\nstdout=%s\nstderr=%s", tc.shell, err, stdout, stderr)
			}
			if len(strings.TrimSpace(stdout)) < 100 {
				t.Fatalf("completion %s output too short:\n%s", tc.shell, stdout)
			}
			if !strings.Contains(stdout, tc.want) {
				t.Fatalf("completion %s output missing %q:\n%s", tc.shell, tc.want, stdout)
			}
		})
	}
}

func TestCompletionCommand_UnsupportedShellErrors(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())

	stdout, stderr, err := executeRootCommandForTest(newCompletionTestRoot(t), "completion", "xonsh")
	if err == nil {
		t.Fatalf("completion xonsh error = nil\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	combined := stdout + stderr + err.Error()
	for _, want := range []string{"unsupported shell", "bash", "zsh", "fish", "powershell"} {
		if !strings.Contains(combined, want) {
			t.Fatalf("completion xonsh missing %q:\nstdout=%s\nstderr=%s\nerr=%v", want, stdout, stderr, err)
		}
	}
}

func TestHermesCLIParityManifestCompletionImplemented(t *testing.T) {
	entry := requireHermesCLIEntry(t, []string{"completion"})
	if entry.Status != hermesCLIImplemented {
		t.Fatalf("completion manifest status = %q, want %q: %+v", entry.Status, hermesCLIImplemented, entry)
	}
	if !strings.Contains(entry.Target, "cmd/gormes completion") {
		t.Fatalf("completion manifest target = %q, want cmd/gormes completion", entry.Target)
	}
}

func newCompletionTestRoot(t *testing.T) *cobra.Command {
	t.Helper()
	failRuntime := func(name string) error {
		t.Fatalf("%s runtime should not run during completion generation", name)
		return fmt.Errorf("%s runtime should not run", name)
	}
	return newRootCommandWithRuntime(rootRuntime{
		runResolvedTUI: func(*cobra.Command, tuiInvocation) error {
			return failRuntime("tui")
		},
		runOneshot: func(*cobra.Command, oneshotInvocation) error {
			return failRuntime("oneshot")
		},
	})
}

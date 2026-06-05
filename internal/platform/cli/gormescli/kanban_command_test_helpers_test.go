package gormescli

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

const Version = "test-version"

type rootRuntime struct{}

func newRootCommandWithRuntime(rootRuntime) *cobra.Command {
	cmd := NewKanbanCommand(KanbanCommandOptions{
		BuildProvenance: func() BuildProvenance {
			return BuildProvenance{Version: Version, GitCommit: "test-git"}
		},
		ExitCodeError: NewExitCodeError,
	})
	cmd.SilenceUsage = true
	return cmd
}

func executeRootCommandForTest(cmd *cobra.Command, args ...string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	err := executeRootCommand(cmd, args...)
	return stdout.String(), stderr.String(), err
}

func executeRootCommand(root *cobra.Command, args ...string) error {
	if len(args) > 0 && args[0] == "kanban" {
		args = args[1:]
	}
	root.SetArgs(args)
	return root.Execute()
}

func exitCodeFromError(err error) int {
	if err == nil {
		return 0
	}
	var coded interface{ ExitCode() int }
	if errors.As(err, &coded) {
		return coded.ExitCode()
	}
	return 1
}

func freshInstallE2EHome(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("GORMES_HOME", filepath.Join(root, "gormes-home"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "xdg-data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "xdg-config"))
	t.Setenv("HERMES_HOME", filepath.Join(root, "hermes-home"))
	t.Setenv("CODEX_HOME", filepath.Join(root, "codex-home"))
	t.Setenv("GORMES_KANBAN_DB", "")
	t.Setenv("GORMES_KANBAN_HOME", "")
	t.Setenv("GORMES_KANBAN_TASK", "")
	t.Setenv("HERMES_KANBAN_BOARD", "")
	t.Setenv("HERMES_KANBAN_DB", "")
	t.Setenv("GORMES_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	return root
}

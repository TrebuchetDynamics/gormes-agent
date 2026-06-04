package admin

import (
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	tuiadmin "github.com/TrebuchetDynamics/gormes-agent/internal/tui/admin"
)

func TestCommandEntriesSkipsHelpAndSorts(t *testing.T) {
	root := &cobra.Command{Use: "gormes"}
	gateway := &cobra.Command{Use: "gateway"}
	gateway.AddCommand(&cobra.Command{Use: "status", Short: "Show gateway status"})
	auth := &cobra.Command{Use: "auth"}
	auth.AddCommand(&cobra.Command{Use: "status <provider>", Short: "Show auth status"})
	root.AddCommand(gateway, auth, &cobra.Command{Use: "hidden", Hidden: true})

	entries := CommandEntries(root)
	paths := make([]string, 0, len(entries))
	byPath := map[string]bool{}
	for _, entry := range entries {
		paths = append(paths, entry.Path)
		byPath[entry.Path] = true
		if entry.Use == "" || !strings.HasPrefix(entry.Use, "gormes ") {
			t.Fatalf("bad Use for entry %#v", entry)
		}
	}
	if byPath["help"] || byPath["hidden"] {
		t.Fatalf("unexpected hidden/help entries: %v", paths)
	}
	if !byPath["auth status"] || !byPath["gateway status"] {
		t.Fatalf("missing expected entries: %v", paths)
	}
}

func TestCommandRunnableAndLabels(t *testing.T) {
	for _, path := range []string{"doctor", "auth status", "gateway status", "kanban list"} {
		if !CommandRunnable(path) {
			t.Fatalf("%s should be runnable", path)
		}
		if CommandRunLabel(path) == "" {
			t.Fatalf("%s should have label", path)
		}
	}
	if CommandRunnable("auth add") || CommandRunLabel("auth add") != "" {
		t.Fatalf("auth add should not be runnable/labeled")
	}
}

func TestCommandRunArgs(t *testing.T) {
	args, label, err := CommandRunArgs("auth status", func() (config.Config, error) {
		cfg := config.Config{}
		cfg.Hermes.Provider = "openai-codex"
		return cfg, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(args, " ") != "auth status openai-codex" || label != "gormes auth status openai-codex" {
		t.Fatalf("args=%v label=%q", args, label)
	}

	_, _, err = CommandRunArgs("auth add", nil)
	if err == nil || !strings.Contains(err.Error(), "not runnable inside gormes admin") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCommandRunnerRejectsMutatingCommandWithoutExecuting(t *testing.T) {
	executed := false
	runner := CommandRunner(RunnerOptions{ExecuteCommand: func(args []string) (string, string, error) {
		executed = true
		return "", "", nil
	}})
	result := runner(tuiadmin.CommandEntry{Path: "auth add"})
	if result.Error == "" {
		t.Fatalf("result error empty: %#v", result)
	}
	if executed {
		t.Fatal("mutating command should not execute")
	}
}

func TestCommandRunnerCapturesOutputAndExitCode(t *testing.T) {
	runner := CommandRunner(RunnerOptions{
		ExecuteCommand: func(args []string) (string, string, error) {
			return "out\n", "err\n", errors.New("boom")
		},
		ExitCode: func(error) int { return 7 },
	})
	result := runner(tuiadmin.CommandEntry{Path: "kanban list"})
	if result.Output != "out\nerr" {
		t.Fatalf("output = %q", result.Output)
	}
	if result.Error != "boom" || result.ExitCode != 7 {
		t.Fatalf("result = %#v", result)
	}
}

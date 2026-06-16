package gormescli

import (
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	tuiadmin "github.com/TrebuchetDynamics/gormes-agent/internal/tui/admin"
)

func TestAdminCommand_NonTTYReturnsRequiresTTYEvidence(t *testing.T) {
	cmd := NewAdminCommand(AdminCommandOptions{})
	cmd.SetIn(strings.NewReader(""))

	stdout, stderr, err := executeRootCommandForTest(cmd)
	if err == nil {
		t.Fatalf("gormes admin error = nil; stdout=%s stderr=%s", stdout, stderr)
	}
	if code := exitCodeFromError(err); code == 0 {
		t.Fatalf("exit code = 0, want non-zero")
	}
	combined := stdout + stderr + err.Error()
	if !strings.Contains(combined, "admin_tui_requires_tty") {
		t.Fatalf("combined output missing admin_tui_requires_tty:\nstdout=%s\nstderr=%s\nerr=%v", stdout, stderr, err)
	}
}

func TestAdminCommandCatalogCoversRootCommandTree(t *testing.T) {
	root := newAdminCatalogRootForTest()
	entries := AdminCommandEntries(root)
	paths := map[string]bool{}
	for _, entry := range entries {
		paths[entry.Path] = true
		if entry.Use == "" {
			t.Fatalf("entry %q has empty Use: %#v", entry.Path, entry)
		}
	}
	for _, want := range []string{
		"admin",
		"auth add",
		"gateway status",
		"kanban list",
		"mcp login",
		"session export",
	} {
		if !paths[want] {
			t.Fatalf("admin command catalog missing %q in %#v", want, entries)
		}
	}
	if paths["help"] {
		t.Fatalf("admin command catalog should not include Cobra help command: %#v", entries)
	}
	for _, removed := range []string{"login", "onboard"} {
		if paths[removed] {
			t.Fatalf("admin command catalog should not include removed top-level command %q in %#v", removed, entries)
		}
	}
}

func TestAdminCommandCatalogMarksOnlySafeCommandsRunnable(t *testing.T) {
	root := newAdminCatalogRootForTest()
	entries := AdminCommandEntries(root)
	byPath := map[string]tuiadmin.CommandEntry{}
	for _, entry := range entries {
		byPath[entry.Path] = entry
	}
	for _, path := range []string{"doctor", "auth status", "gateway status", "kanban list"} {
		entry, ok := byPath[path]
		if !ok {
			t.Fatalf("missing command catalog entry %q", path)
		}
		if !entry.Runnable {
			t.Fatalf("entry %q should be runnable in admin TUI: %#v", path, entry)
		}
		if entry.RunLabel == "" {
			t.Fatalf("entry %q has empty RunLabel: %#v", path, entry)
		}
	}
	for _, path := range []string{"auth add", "gateway stop", "kanban create"} {
		entry, ok := byPath[path]
		if !ok {
			t.Fatalf("missing command catalog entry %q", path)
		}
		if entry.Runnable {
			t.Fatalf("entry %q should not be runnable in admin TUI: %#v", path, entry)
		}
	}
}

func TestAdminCommandRunnerExecutesSafeCommandAndRejectsMutating(t *testing.T) {
	var executedArgs []string
	runner := AdminCommandRunner(AdminRunnerOptions{
		ExecuteCommand: func(args []string) (string, string, error) {
			executedArgs = append([]string(nil), args...)
			return "No Kanban tasks\n", "", nil
		},
		ExitCode: exitCodeFromError,
	})

	result := runner(tuiadmin.CommandEntry{Path: "kanban list", RunLabel: "gormes kanban list", Runnable: true})
	if result.Error != "" {
		t.Fatalf("kanban list result error = %q; output=%s", result.Error, result.Output)
	}
	if !reflect.DeepEqual(executedArgs, []string{"kanban", "list"}) {
		t.Fatalf("executed args = %#v, want kanban list", executedArgs)
	}
	if !strings.Contains(result.Output, "No Kanban tasks") {
		t.Fatalf("kanban list output = %q, want empty-board summary", result.Output)
	}

	result = runner(tuiadmin.CommandEntry{Path: "auth add", Use: "gormes auth add <provider>"})
	if result.Error == "" {
		t.Fatalf("auth add result error empty; result=%#v", result)
	}
	if !strings.Contains(result.Error, "not runnable inside gormes admin") {
		t.Fatalf("auth add error = %q", result.Error)
	}
}

func TestAdminCommandRunnerExecutesSafeKanbanCommandThroughCLI(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	runner := AdminCommandRunner(AdminRunnerOptions{
		ExecuteCommand: func(args []string) (string, string, error) {
			return executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), args...)
		},
		ExitCode: exitCodeFromError,
	})

	result := runner(tuiadmin.CommandEntry{Path: "kanban list", RunLabel: "gormes kanban list", Runnable: true})
	if result.Error != "" {
		t.Fatalf("kanban list result error = %q; output=%s", result.Error, result.Output)
	}
	if !strings.Contains(result.Output, "No Kanban tasks") {
		t.Fatalf("kanban list output = %q, want empty-board summary", result.Output)
	}
}

func TestAdminCommandRunArgsResolveConfiguredAuthProvider(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	writeOneshotFlagConfig(t, []byte("[hermes]\nprovider = \"openai-codex\"\n"))

	args, label, err := AdminCommandRunArgs("auth status")
	if err != nil {
		t.Fatalf("adminCommandRunArgs(auth status): %v", err)
	}
	got := strings.Join(args, " ")
	if got != "auth status openai-codex" {
		t.Fatalf("args = %q, want auth status openai-codex", got)
	}
	if label != "gormes auth status openai-codex" {
		t.Fatalf("label = %q, want gormes auth status openai-codex", label)
	}
}

func newAdminCatalogRootForTest() *cobra.Command {
	factories := stubRootFactories()
	factories["admin"] = func() *cobra.Command { return NewAdminCommand(AdminCommandOptions{}) }
	factories["auth"] = func() *cobra.Command {
		cmd := &cobra.Command{Use: "auth"}
		cmd.AddCommand(
			&cobra.Command{Use: "add <provider>", Short: "Add provider credentials"},
			&cobra.Command{Use: "status <provider>", Short: "Show auth status"},
		)
		return cmd
	}
	factories["doctor"] = func() *cobra.Command { return &cobra.Command{Use: "doctor", Short: "Check local readiness"} }
	factories["gateway"] = func() *cobra.Command {
		cmd := &cobra.Command{Use: "gateway"}
		cmd.AddCommand(
			&cobra.Command{Use: "status", Short: "Show gateway status"},
			&cobra.Command{Use: "stop", Short: "Stop gateway"},
		)
		return cmd
	}
	factories["kanban"] = func() *cobra.Command {
		cmd := &cobra.Command{Use: "kanban"}
		cmd.AddCommand(
			&cobra.Command{Use: "list", Short: "List Kanban tasks"},
			&cobra.Command{Use: "create", Short: "Create a Kanban task"},
		)
		return cmd
	}
	factories["mcp"] = func() *cobra.Command {
		cmd := &cobra.Command{Use: "mcp"}
		cmd.AddCommand(&cobra.Command{Use: "login <server>", Short: "Refresh OAuth"})
		return cmd
	}
	factories["session"] = func() *cobra.Command {
		cmd := &cobra.Command{Use: "session"}
		cmd.AddCommand(&cobra.Command{Use: "export <id>", Short: "Export a session"})
		return cmd
	}
	return NewRootCommand(RootOptions{Version: Version}, factories)
}

package main

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/admin"
)

func TestAdminCommand_NonTTYReturnsRequiresTTYEvidence(t *testing.T) {
	cmd := newRootCommandWithRuntime(rootRuntime{})
	cmd.SetIn(strings.NewReader(""))

	stdout, stderr, err := executeRootCommandForTest(cmd, "admin")
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
	root := newRootCommandWithRuntime(rootRuntime{})
	entries := adminCommandEntries(root)
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
		"login",
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
}

func TestAdminCommandCatalogMarksOnlySafeCommandsRunnable(t *testing.T) {
	root := newRootCommandWithRuntime(rootRuntime{})
	entries := adminCommandEntries(root)
	byPath := map[string]admin.CommandEntry{}
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
	setupOneshotFlagTestEnv(t)
	runner := newAdminCommandRunner()

	result := runner(admin.CommandEntry{Path: "kanban list", RunLabel: "gormes kanban list", Runnable: true})
	if result.Error != "" {
		t.Fatalf("kanban list result error = %q; output=%s", result.Error, result.Output)
	}
	if !strings.Contains(result.Output, "No Kanban tasks") {
		t.Fatalf("kanban list output = %q, want empty-board summary", result.Output)
	}

	result = runner(admin.CommandEntry{Path: "auth add", Use: "gormes auth add <provider>"})
	if result.Error == "" {
		t.Fatalf("auth add result error empty; result=%#v", result)
	}
	if !strings.Contains(result.Error, "not runnable inside gormes admin") {
		t.Fatalf("auth add error = %q", result.Error)
	}
}

func TestAdminCommandRunArgsResolveConfiguredAuthProvider(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	writeOneshotFlagConfig(t, []byte("[hermes]\nprovider = \"openai-codex\"\n"))

	args, label, err := adminCommandRunArgs("auth status")
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

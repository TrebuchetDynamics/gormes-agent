package main

import (
	"strings"
	"testing"
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

package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func executeRootCommandForTest(cmd *cobra.Command, args ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	err := executeRootCommand(cmd, args...)
	return stdout.String(), stderr.String(), err
}

func TestGormesLoginUnknownCommandEmitsTypoSuggestion(t *testing.T) {
	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeRootCommandForTest(cmd, "login")
	if err == nil {
		t.Fatalf("gormes login error = nil; stdout=%s stderr=%s", stdout, stderr)
	}
	if code := exitCodeFromError(err); code == 0 {
		t.Fatalf("exit code = 0, want non-zero")
	}
	combined := stdout + stderr + err.Error()
	want := "did you mean \"gormes auth add <provider> --type oauth\"?"
	if !strings.Contains(combined, want) {
		t.Fatalf("combined output missing suggestion %q:\nstdout=%s\nstderr=%s\nerr=%v", want, stdout, stderr, err)
	}
}

func TestGormesLoginWithProviderFlagsEmitsSuggestion(t *testing.T) {
	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeRootCommandForTest(cmd, "login", "--provider", "openai-codex", "--no-browser")
	if err == nil {
		t.Fatalf("gormes login --provider error = nil; stdout=%s stderr=%s", stdout, stderr)
	}
	combined := stdout + stderr + err.Error()
	want := "did you mean \"gormes auth add <provider> --type oauth\"?"
	if !strings.Contains(combined, want) {
		t.Fatalf("combined output missing suggestion %q:\nstdout=%s\nstderr=%s\nerr=%v", want, stdout, stderr, err)
	}
}

func TestGormesLoginIsNotRegisteredAsCobraCommand(t *testing.T) {
	cmd := newRootCommandWithRuntime(rootRuntime{})
	found, _, err := cmd.Find([]string{"login"})
	if err == nil && found != nil && found.Name() == "login" {
		t.Fatalf("gormes login is registered as Cobra command: %+v", found)
	}
}

func TestGormesLoginSuggestionParityWithMigrateTypo(t *testing.T) {
	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeRootCommandForTest(cmd, "login")
	if err == nil {
		t.Fatalf("gormes login error = nil; stdout=%s stderr=%s", stdout, stderr)
	}
	combined := stdout + stderr + err.Error()
	for _, want := range []string{"unknown command", "did you mean", "auth add <provider> --type oauth"} {
		if !strings.Contains(combined, want) {
			t.Fatalf("combined output missing %q:\nstdout=%s\nstderr=%s\nerr=%v", want, stdout, stderr, err)
		}
	}
	if strings.Contains(combined, "auth_oauth_saved") || strings.Contains(combined, "auth_added") {
		t.Fatalf("login suggestion appears to perform auth side effects:\n%s", combined)
	}
}

func TestGormesLoginSuggestionRedactsProviderArgValues(t *testing.T) {
	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeRootCommandForTest(cmd, "login", "--provider", "plain-secret-provider")
	if err == nil {
		t.Fatalf("gormes login --provider secret error = nil; stdout=%s stderr=%s", stdout, stderr)
	}
	combined := stdout + stderr + err.Error()
	if strings.Contains(combined, "plain-secret-provider") {
		t.Fatalf("login suggestion leaked provider arg value:\n%s", combined)
	}
	if !strings.Contains(combined, "auth add <provider> --type oauth") {
		t.Fatalf("combined output missing redacted provider placeholder:\n%s", combined)
	}
}

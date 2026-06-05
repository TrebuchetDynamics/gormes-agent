package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

type sendCommandResult = gormescli.SendResult

func TestSendCommandPositionalMessageDoesNotStartTUI(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	var gotTarget, gotMessage string
	cmd := newRootCommandWithRuntime(rootRuntime{
		sendMessage: func(_ context.Context, target, message string) (sendCommandResult, error) {
			gotTarget = target
			gotMessage = message
			return sendCommandResult{Success: true, Target: target, MessageID: "m123"}, nil
		},
		runResolvedTUI: func(*cobra.Command, tuiInvocation) error {
			t.Fatal("send command must not start TUI")
			return nil
		},
	})

	stdout, stderr, err := executeRootCommandForTest(cmd, "send", "--to", "telegram", "hello world")
	if err != nil {
		t.Fatalf("send error = %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if gotTarget != "telegram" || gotMessage != "hello world" {
		t.Fatalf("send target/message = %q/%q, want telegram/hello world", gotTarget, gotMessage)
	}
	if !strings.Contains(stdout, "sent") {
		t.Fatalf("stdout = %q, want sent banner", stdout)
	}
}

func TestSendCommandReadsFileAndPreservesNewline(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	body := filepath.Join(t.TempDir(), "msg.txt")
	if err := os.WriteFile(body, []byte("from a file\n"), 0o600); err != nil {
		t.Fatalf("write body: %v", err)
	}

	var gotMessage string
	cmd := newRootCommandWithRuntime(rootRuntime{
		sendMessage: func(_ context.Context, target, message string) (sendCommandResult, error) {
			gotMessage = message
			return sendCommandResult{Success: true, Target: target}, nil
		},
	})
	stdout, stderr, err := executeRootCommandForTest(cmd, "send", "--to", "slack:#eng", "--file", body)
	if err != nil {
		t.Fatalf("send error = %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if gotMessage != "from a file\n" {
		t.Fatalf("message = %q, want file body with trailing newline", gotMessage)
	}
}

func TestSendCommandReadsPipedStdin(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	var gotMessage string
	cmd := newRootCommandWithRuntime(rootRuntime{
		sendMessage: func(_ context.Context, target, message string) (sendCommandResult, error) {
			gotMessage = message
			return sendCommandResult{Success: true, Target: target}, nil
		},
	})
	cmd.SetIn(strings.NewReader("piped body\n"))
	stdout, stderr, err := executeRootCommandForTest(cmd, "send", "--to", "discord:#ops")
	if err != nil {
		t.Fatalf("send error = %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if gotMessage != "piped body\n" {
		t.Fatalf("message = %q, want stdin body with trailing newline", gotMessage)
	}
}

func TestSendCommandRejectsInvalidFileText(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	body := filepath.Join(t.TempDir(), "bad.bin")
	if err := os.WriteFile(body, []byte{0xff, 0xfe, 0x00}, 0o600); err != nil {
		t.Fatalf("write body: %v", err)
	}

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "send", "--to", "telegram", "--file", body)
	if err == nil {
		t.Fatalf("send error = nil\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if code := exitCodeFromError(err); code != 2 {
		t.Fatalf("exit code = %d, want 2; err=%v stderr=%s", code, err, stderr)
	}
	combined := stdout + stderr + err.Error()
	if !strings.Contains(combined, "cannot read") || !strings.Contains(combined, "message body must be UTF-8 text") {
		t.Fatalf("invalid file output = %q, want stable cannot-read text error", combined)
	}
	if strings.Contains(combined, "\xff") || strings.Contains(combined, "\xfe") {
		t.Fatalf("invalid file output leaked raw bytes: %q", combined)
	}
}

func TestSendCommandDryRunJSONSanitizesTerminalResponses(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "send", "--to", "telegram", "--dry-run", "--json", "hello\x1b[53;1Rworld")
	if err != nil {
		t.Fatalf("dry-run error = %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var got sendCommandResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode dry-run JSON %q: %v", stdout, err)
	}
	if !got.Success || !got.Skipped || !got.DryRun {
		t.Fatalf("dry-run result = %+v, want success skipped dry_run", got)
	}
	if got.Message != "helloworld" {
		t.Fatalf("dry-run message = %q, want sanitized text", got.Message)
	}
	if strings.Contains(stdout, "\x1b[53;1R") {
		t.Fatalf("dry-run JSON leaked terminal control response: %q", stdout)
	}
}

func TestSendCommandDryRunJSONSanitizesSubjectAndTarget(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "send", "--to", "telegram\x1b[53;1R", "--subject", "[CI]\x1b[53;1R", "--dry-run", "--json", "body")
	if err != nil {
		t.Fatalf("dry-run error = %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var got sendCommandResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode dry-run JSON %q: %v", stdout, err)
	}
	if got.Target != "telegram" {
		t.Fatalf("dry-run target = %q, want sanitized target", got.Target)
	}
	if got.Message != "[CI]\n\nbody" {
		t.Fatalf("dry-run message = %q, want sanitized subject and body", got.Message)
	}
	if strings.Contains(stdout, "\x1b[53;1R") {
		t.Fatalf("dry-run JSON leaked terminal control response: %q", stdout)
	}
}

func TestSendCommandListsChannelDirectory(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	home := config.GormesHome()
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}
	dir := `{"platforms":{"telegram":[{"id":"-100123","name":"Ops","type":"group"}],"discord":[{"id":"555","name":"general"}]}}`
	if err := os.WriteFile(filepath.Join(home, "channel_directory.json"), []byte(dir), 0o600); err != nil {
		t.Fatalf("write directory: %v", err)
	}

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "send", "--list", "telegram")
	if err != nil {
		t.Fatalf("send --list error = %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "telegram:Ops") || strings.Contains(stdout, "discord") {
		t.Fatalf("filtered list output = %q, want only telegram targets", stdout)
	}
}

func TestSendCommandDefaultBackendUnavailable(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "send", "--to", "telegram", "hello")
	if err == nil {
		t.Fatalf("send error = nil\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if code := exitCodeFromError(err); code != 1 {
		t.Fatalf("exit code = %d, want 1; err=%v stderr=%s", code, err, stderr)
	}
	if !strings.Contains(stderr+err.Error(), "send_backend_unavailable") {
		t.Fatalf("output = stdout:%q stderr:%q err:%v, want typed backend unavailable", stdout, stderr, err)
	}
}

func TestSendCommandBackendErrorIsSanitized(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommandWithRuntime(rootRuntime{
		sendMessage: func(context.Context, string, string) (sendCommandResult, error) {
			return sendCommandResult{}, errors.New("platform failed \x1b[53;1R")
		},
	})
	stdout, stderr, err := executeRootCommandForTest(cmd, "send", "--to", "telegram", "hello")
	if err == nil {
		t.Fatalf("send error = nil\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if strings.Contains(stderr+err.Error(), "\x1b[53;1R") {
		t.Fatalf("backend error leaked terminal control response: stderr=%q err=%v", stderr, err)
	}
}

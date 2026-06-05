package gateway

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTermuxGatewayStatusTextIncludesForegroundLifecycleGuidance(t *testing.T) {
	setupTermuxGatewayLifecycleTestEnv(t)

	stdout, stderr, err := executeGatewayStatusCommand(t)
	if err != nil {
		t.Fatalf("gateway status: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Termux gateway:",
		"foreground/tmux lifecycle",
		"gormes gateway",
		"termux-wake-lock",
		"Termux notification:",
		"available",
		"Android battery",
		"best-effort",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("gateway status missing Termux guidance %q:\n%s", want, stdout)
		}
	}
	for _, forbidden := range []string{"service_restart", "Scheduled Task", "systemd"} {
		if strings.Contains(stdout, forbidden) {
			t.Fatalf("Termux gateway status advertised desktop service assumption %q:\n%s", forbidden, stdout)
		}
	}
}

func TestTermuxGatewayStartGuidesToForegroundTmuxWithoutServiceAssumptions(t *testing.T) {
	setupTermuxGatewayLifecycleTestEnv(t)

	stdout, stderr, err := executeGatewayMutatingCommand(t, "start")
	if err == nil {
		t.Fatalf("gateway start under Termux should be guidance-only; stdout=%s stderr=%s", stdout, stderr)
	}
	message := err.Error() + "\n" + stdout + "\n" + stderr
	for _, want := range []string{
		"Termux",
		"run `gormes gateway` inside tmux",
		"termux-wake-lock",
		"Android battery",
		"gateway status",
		"gateway stop",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("gateway start guidance missing %q:\n%s", want, message)
		}
	}
	for _, forbidden := range []string{"service_restart", "Scheduled Task", "systemd"} {
		if strings.Contains(message, forbidden) {
			t.Fatalf("Termux gateway start advertised desktop service assumption %q:\n%s", forbidden, message)
		}
	}
	assertGatewayStopDidNotOpenDurableStores(t)
}

func TestTermuxGatewayStatusJSONKeepsDesktopContract(t *testing.T) {
	setupTermuxGatewayLifecycleTestEnv(t)

	stdout, stderr, err := executeGatewayStatusCommand(t, "--json")
	if err != nil {
		t.Fatalf("gateway status --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var got gatewayStatusJSON
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("gateway status --json must remain parseable JSON: %v\n%s", jsonErr, stdout)
	}
	if got.Runtime.Platforms == nil {
		t.Fatalf("runtime.platforms = nil, want empty map for existing JSON contract")
	}
	if strings.Contains(stdout, "Termux gateway:") || strings.Contains(stdout, "foreground/tmux lifecycle") {
		t.Fatalf("gateway status --json mixed human Termux guidance into JSON:\n%s", stdout)
	}
	if strings.Contains(stdout, "Termux notification:") {
		t.Fatalf("gateway status --json mixed human Termux notification guidance into JSON:\n%s", stdout)
	}
}

func TestTermuxGatewayStopJSONKeepsDesktopContract(t *testing.T) {
	setupTermuxGatewayLifecycleTestEnv(t)

	stdout, stderr, err := executeGatewayMutatingCommand(t, "stop", "--json", "--timeout=0")
	if err != nil {
		t.Fatalf("gateway stop --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var got gatewayStopReportJSON
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("gateway stop --json must remain parseable JSON: %v\n%s", jsonErr, stdout)
	}
	if got.Action != "noop" || got.Live {
		t.Fatalf("gateway stop --json = %+v, want noop live=false with missing runtime", got)
	}
	if strings.Contains(stdout, "Termux gateway:") || strings.Contains(stdout, "foreground/tmux lifecycle") {
		t.Fatalf("gateway stop --json mixed human Termux guidance into JSON:\n%s", stdout)
	}
}

func TestTermuxGatewayStatusTextReportsNotificationUnavailable(t *testing.T) {
	setupTermuxGatewayLifecycleTestEnvWithCommands(t, []string{"tmux", "termux-wake-lock"})

	stdout, stderr, err := executeGatewayStatusCommand(t)
	if err != nil {
		t.Fatalf("gateway status: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Termux notification:",
		"optional_notification_unavailable",
		"termux-notification missing",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("gateway status missing notification degradation %q:\n%s", want, stdout)
		}
	}
}

func setupTermuxGatewayLifecycleTestEnv(t *testing.T) {
	setupTermuxGatewayLifecycleTestEnvWithCommands(t, []string{"tmux", "termux-wake-lock", "termux-notification"})
}

func setupTermuxGatewayLifecycleTestEnvWithCommands(t *testing.T, commands []string) {
	t.Helper()
	root, err := os.MkdirTemp("/tmp", "gormes-termux-gateway-")
	if err != nil {
		t.Fatalf("create Termux gateway fixture root: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })

	prefix := filepath.Join(root, "com.termux", "files", "usr")
	binDir := filepath.Join(prefix, "bin")
	for _, dir := range []string{
		filepath.Join(root, "home"),
		filepath.Join(root, "gormes"),
		filepath.Join(root, "data"),
		filepath.Join(root, "config"),
		filepath.Join(root, "hermes"),
		binDir,
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	for _, name := range commands {
		if err := os.WriteFile(filepath.Join(binDir, name), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	t.Setenv("TERMUX_VERSION", "0.119.0")
	t.Setenv("PREFIX", prefix)
	t.Setenv("PATH", binDir)
	t.Setenv("HOME", filepath.Join(root, "home"))
	t.Setenv("GORMES_HOME", filepath.Join(root, "gormes"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("HERMES_HOME", filepath.Join(root, "hermes"))
}

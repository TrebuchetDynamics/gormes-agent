package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/spf13/cobra"
)

func TestSetupGatewayChecklistShowsCorePlatforms(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	fake := &setupCommandFakeSeams{isTTY: false}
	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "gateway", "--non-interactive")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}

	for _, want := range []string{
		"Messaging Platforms",
		"Which platforms would you like to set up?",
		"Telegram",
		"telegram",
		"Discord",
		"discord",
		"Slack",
		"slack",
		"not configured",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if _, err := os.Stat(config.ConfigPath()); !os.IsNotExist(err) {
		t.Fatalf("non-interactive setup gateway mutated config path %s: %v", config.ConfigPath(), err)
	}
}

func TestSetupGatewayPreselectsConfiguredPlatforms(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	writeSetupGatewayFixtureConfig(t, `
[telegram]
bot_token = "123456:test-token"
allowed_chat_id = 4242

[discord]
token = "discord-token"
allowed_channel_id = "D42"

[slack]
enabled = true
bot_token = "xoxb-test"
app_token = "xapp-test"
allowed_channel_id = "C42"
`)

	fake := &setupCommandFakeSeams{isTTY: false}
	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "gateway", "--non-interactive")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"[x] Telegram",
		"telegram",
		"configured (allowed_chat_id=4242)",
		"[x] Discord",
		"discord",
		"configured (allowed_channel_id=D42)",
		"[x] Slack",
		"slack",
		"configured (allowed_channel_id=C42)",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestSetupGatewayNoSelectionDoesNotMutateConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	fake := &setupCommandFakeSeams{isTTY: true}
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "\n", "gateway")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Messaging platforms (comma-separated numbers or ids, blank to keep current):",
		"No platform setup changes selected.",
		"Keeping current gateway platform configuration.",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout+stderr, "setup_gateway_platform_row_backed") {
		t.Fatalf("blank selection dispatched platform setup:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if _, err := os.Stat(config.ConfigPath()); !os.IsNotExist(err) {
		t.Fatalf("blank setup gateway mutated config path %s: %v", config.ConfigPath(), err)
	}
}

func TestSetupGatewaySelectedPlatformDelegatesOrReportsRowBacked(t *testing.T) {
	t.Run("row backed by default", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("GORMES_HOME", home)

		fake := &setupCommandFakeSeams{isTTY: true}
		stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "telegram\n", "gateway")
		if err != nil {
			t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
		}
		for _, want := range []string{
			"setup_gateway_platform_row_backed: platform=telegram recommended_command=\"gormes setup gateway\"",
			"Start messaging with: gormes gateway",
		} {
			if !strings.Contains(stdout, want) {
				t.Fatalf("stdout missing %q:\n%s", want, stdout)
			}
		}
	})

	t.Run("injected platform handlers", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("GORMES_HOME", home)

		var called []string
		fake := &setupCommandFakeSeams{isTTY: true}
		fake.runGatewayPlatform = func(cmd *cobra.Command, platform string) error {
			called = append(called, platform)
			cmd.Printf("setup_gateway_platform_delegated: platform=%s\n", platform)
			return nil
		}
		stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "1,slack\n", "gateway")
		if err != nil {
			t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
		}
		if strings.Join(called, ",") != "telegram,slack" {
			t.Fatalf("called platforms = %v, want telegram,slack", called)
		}
		for _, want := range []string{
			"setup_gateway_platform_delegated: platform=telegram",
			"setup_gateway_platform_delegated: platform=slack",
		} {
			if !strings.Contains(stdout, want) {
				t.Fatalf("stdout missing %q:\n%s", want, stdout)
			}
		}
	})
}

func TestSetupGatewayDoesNotStartGateway(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	fake := &setupCommandFakeSeams{isTTY: true}
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "slack\n", "gateway")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, path := range []string{
		config.SessionDBPath(),
		config.MemoryDBPath(),
		config.GatewayRuntimeStatusPath(),
	} {
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("setup gateway opened runtime/startup artifact %s\nstdout=%s", path, stdout)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat runtime/startup artifact %s: %v", path, err)
		}
	}
}

func TestSetupGatewaySectionRoutesThroughGatewaySeam(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	gatewayCalls := 0
	fake := &setupCommandFakeSeams{isTTY: true}
	fake.runSetupGateway = func(cmd *cobra.Command, nonInteractive bool) error {
		gatewayCalls++
		if nonInteractive {
			t.Fatal("interactive gateway setup was marked non-interactive")
		}
		cmd.Println("gateway seam reached")
		return nil
	}

	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "gateway")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if gatewayCalls != 1 {
		t.Fatalf("RunSetupGateway calls = %d, want 1", gatewayCalls)
	}
	if !strings.Contains(stdout, "gateway seam reached") {
		t.Fatalf("stdout missing gateway seam output:\n%s", stdout)
	}
}

func writeSetupGatewayFixtureConfig(t *testing.T, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(config.ConfigPath()), 0o700); err != nil {
		t.Fatalf("mkdir config home: %v", err)
	}
	if err := os.WriteFile(config.ConfigPath(), []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

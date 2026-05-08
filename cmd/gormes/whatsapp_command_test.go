package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestWhatsAppTopLevelCommandRendersPairingPreflight(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "whatsapp", "--plan", "--mode", "bot", "--allowed-users", "528112345678,528187654321")
	if err != nil {
		t.Fatalf("gormes whatsapp error = %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	for _, want := range []string{
		"WhatsApp pairing setup",
		"Baileys bridge",
		"WhatsApp Web",
		"not the official Meta Business API",
		"Mode: bot",
		"Config file:  " + config.ConfigPath(),
		"Secrets file: " + config.EnvPath(),
		"Session dir:  " + filepath.Join(home, "whatsapp", "session"),
		"Bridge log:   " + filepath.Join(home, "whatsapp", "bridge.log"),
		"WHATSAPP_ENABLED=true",
		"WHATSAPP_MODE=bot",
		"WHATSAPP_ALLOWED_USERS=528112345678,528187654321",
		"node scripts/whatsapp-bridge/bridge.js --port 3000 --session " + filepath.Join(home, "whatsapp", "session") + " --mode bot",
		"Run without --plan to start the live QR pairing wizard.",
		"gormes gateway status",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	for _, forbidden := range []string{
		"hermes whatsapp",
		"hermes gateway",
		"~/.hermes",
		"/.hermes/",
	} {
		if strings.Contains(stdout, forbidden) {
			t.Fatalf("stdout contains Hermes-owned text %q:\n%s", forbidden, stdout)
		}
	}
}

func TestWhatsAppCommandRunsPairingWizardWithFakeBridgeEvents(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	if err := os.WriteFile(config.EnvPath(), []byte("WHATSAPP_ENABLED=true\nWHATSAPP_ALLOWED_USERS=5218112554202,5218116750683\n"), 0o600); err != nil {
		t.Fatalf("write env fixture: %v", err)
	}

	cmd := newWhatsAppCommandWithSeams(whatsappCommandSeams{
		InstallBridgeDependencies: func(_ context.Context, _ string, out io.Writer) error {
			_, err := io.WriteString(out, "✓ Bridge dependencies already installed\n")
			return err
		},
		RunBridgePairing: func(_ context.Context, _ whatsappPairingPlan, out io.Writer) error {
			_, err := io.WriteString(out, "\n📱 Scan this QR code with WhatsApp on your phone:\n\n▄▄FAKE-QR\n\nWaiting for scan...\n\n{\"level\":50,\"msg\":\"stream errored out\",\"fullErrorNode\":{\"tag\":\"stream:error\",\"attrs\":{\"code\":\"515\"}}}\n↻ WhatsApp requested restart (code 515). Reconnecting...\n✅ WhatsApp connected!\n✅ Pairing complete. Credentials saved.\n")
			return err
		},
	})

	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--mode", "bot"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("gormes whatsapp error = %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	out := stdout.String()
	for _, want := range []string{
		"⚕ WhatsApp Setup",
		"✓ Mode: separate bot number",
		"✓ WhatsApp is already enabled",
		"✓ Allowed users: 5218112554202,5218116750683",
		"✓ Bridge dependencies already installed",
		"Settings → Linked Devices → Link a Device",
		"📱 WhatsApp pairing mode",
		"📁 Session: " + filepath.Join(home, "whatsapp", "session"),
		"▄▄FAKE-QR",
		"Waiting for scan...",
		"↻ WhatsApp requested restart (code 515). Reconnecting...",
		"✅ WhatsApp connected!",
		"✅ Pairing complete. Credentials saved.",
		"✓ WhatsApp paired successfully!",
		"Start the gateway:  gormes gateway",
		"Tip: Agent responses are prefixed with '⚕ Gormes Agent'",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q:\n%s", want, out)
		}
	}
	for _, forbidden := range []string{
		"hermes whatsapp",
		"hermes gateway",
		"Hermes Agent",
		"~/.hermes",
		"/.hermes/",
	} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("stdout contains Hermes-owned text %q:\n%s", forbidden, out)
		}
	}

	envData, err := os.ReadFile(config.EnvPath())
	if err != nil {
		t.Fatalf("read persisted env: %v", err)
	}
	for _, want := range []string{
		"WHATSAPP_ENABLED=true",
		"WHATSAPP_MODE=bot",
		"WHATSAPP_ALLOWED_USERS=5218112554202,5218116750683",
	} {
		if !strings.Contains(string(envData), want) {
			t.Fatalf("env missing %q:\n%s", want, string(envData))
		}
	}
}

func TestWhatsAppCommandRejectsInvalidMode(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "whatsapp", "--mode", "business-api")
	if err == nil {
		t.Fatalf("gormes whatsapp invalid mode error = nil\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if code := exitCodeFromError(err); code != 2 {
		t.Fatalf("exit code = %d, want 2; err=%v", code, err)
	}
	if !strings.Contains(err.Error(), `whatsapp: unsupported account mode "business-api"`) {
		t.Fatalf("err = %v, want unsupported mode evidence", err)
	}
}

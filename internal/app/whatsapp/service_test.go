package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestCommandPlanJSONEmitsStructuredPlan(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	cmd := NewCommand(Options{BuildProvenance: func() BuildProvenance { return BuildProvenance{Version: "test-version", GitCommit: "test-commit"} }})
	stdout, stderr, err := executeCommand(cmd, "--plan", "--json", "--mode", "bot", "--allowed-users", "528112345678,528187654321", "--debug")
	if err != nil {
		t.Fatalf("whatsapp --plan --json error = %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}

	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Mode          string   `json:"mode"`
		ConfigPath    string   `json:"config_path"`
		EnvPath       string   `json:"env_path"`
		SessionDir    string   `json:"session_dir"`
		BridgeLog     string   `json:"bridge_log"`
		AllowedUsers  string   `json:"allowed_users"`
		Debug         bool     `json:"debug"`
		AllowAllUsers bool     `json:"allow_all_users"`
		BridgeCommand []string `json:"bridge_command"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("whatsapp --plan --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != "test-version" || got.Build.GitCommit != "test-commit" {
		t.Fatalf("build = %+v, want injected provenance", got.Build)
	}
	if got.Mode != "bot" {
		t.Errorf("mode = %q, want bot", got.Mode)
	}
	if got.ConfigPath != config.ConfigPath() || got.EnvPath != config.EnvPath() {
		t.Errorf("paths = %q/%q, want config/env paths", got.ConfigPath, got.EnvPath)
	}
	if got.SessionDir != filepath.Join(home, "whatsapp", "session") {
		t.Errorf("session_dir = %q", got.SessionDir)
	}
	if got.BridgeLog != filepath.Join(home, "whatsapp", "bridge.log") {
		t.Errorf("bridge_log = %q", got.BridgeLog)
	}
	if got.AllowedUsers != "528112345678,528187654321" || !got.Debug || len(got.BridgeCommand) == 0 {
		t.Errorf("unexpected report: %+v", got)
	}
}

func TestCommandRunsPairingWizardWithFakeBridgeEvents(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	if err := os.WriteFile(config.EnvPath(), []byte("WHATSAPP_ENABLED=true\nWHATSAPP_ALLOWED_USERS=5218112554202,5218116750683\n"), 0o600); err != nil {
		t.Fatalf("write env fixture: %v", err)
	}

	cmd := NewCommandWithSeams(Seams{
		InstallBridgeDependencies: func(_ context.Context, _ string, out io.Writer) error {
			_, err := io.WriteString(out, "✓ Bridge dependencies already installed\n")
			return err
		},
		RunBridgePairing: func(_ context.Context, _ PairingPlan, out io.Writer) error {
			_, err := io.WriteString(out, "\n📱 Scan this QR code with WhatsApp on your phone:\n\n▄▄FAKE-QR\n\nWaiting for scan...\n\n↻ WhatsApp requested restart (code 515). Reconnecting...\n✅ WhatsApp connected!\n✅ Pairing complete. Credentials saved.\n")
			return err
		},
	}, Options{})

	stdout, stderr, err := executeCommand(cmd, "--mode", "bot")
	if err != nil {
		t.Fatalf("whatsapp error = %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	for _, want := range []string{
		"⚕ WhatsApp Setup",
		"✓ Mode: separate bot number",
		"✓ WhatsApp is already enabled",
		"✓ Allowed users: 5218112554202,5218116750683",
		"✓ Bridge dependencies already installed",
		"Settings → Linked Devices → Link a Device",
		"▄▄FAKE-QR",
		"✓ WhatsApp paired successfully!",
		"Start the gateway:  gormes gateway",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}

	envData, err := os.ReadFile(config.EnvPath())
	if err != nil {
		t.Fatalf("read persisted env: %v", err)
	}
	for _, want := range []string{"WHATSAPP_ENABLED=true", "WHATSAPP_MODE=bot", "WHATSAPP_ALLOWED_USERS=5218112554202,5218116750683"} {
		if !strings.Contains(string(envData), want) {
			t.Fatalf("env missing %q:\n%s", want, string(envData))
		}
	}
}

func TestCommandRejectsInvalidModeWithExitCode(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	stdout, stderr, err := executeCommand(NewCommand(Options{}), "--mode", "business-api")
	if err == nil {
		t.Fatalf("invalid mode error = nil\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	coded, ok := err.(interface{ ExitCode() int })
	if !ok || coded.ExitCode() != 2 {
		t.Fatalf("exit code = %#v, want 2; err=%v", coded, err)
	}
	if !strings.Contains(err.Error(), `whatsapp: unsupported account mode "business-api"`) {
		t.Fatalf("err = %v, want unsupported mode evidence", err)
	}
}

func executeCommand(cmd interface {
	SetOut(io.Writer)
	SetErr(io.Writer)
	SetArgs([]string)
	Execute() error
}, args ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

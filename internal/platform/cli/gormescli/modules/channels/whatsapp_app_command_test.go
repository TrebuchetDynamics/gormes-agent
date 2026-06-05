package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

const testWhatsAppCommandVersion = "test-version"

func whatsappCommandOptions() WhatsAppAppOptions {
	return WhatsAppAppOptions{BuildProvenance: func() WhatsAppBuildProvenance {
		return WhatsAppBuildProvenance{Version: testWhatsAppCommandVersion, GitCommit: "test-git"}
	}}
}

func newWhatsAppCommandForTest() *cobra.Command {
	return NewWhatsAppAppCommand(whatsappCommandOptions())
}

type whatsappCommandSeams = WhatsAppAppSeams
type whatsappPairingPlan = WhatsAppPairingPlan

func newWhatsAppCommandWithSeams(seams whatsappCommandSeams) *cobra.Command {
	return NewWhatsAppAppCommandWithSeams(seams, whatsappCommandOptions())
}

func TestWhatsAppTopLevelCommandRendersPairingPreflight(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	stdout, stderr, err := executeWhatsAppCommandForTest(newWhatsAppCommandForTest(), "--plan", "--mode", "bot", "--allowed-users", "528112345678,528187654321")
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
		"gormes channels --channel whatsapp",
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

// TestWhatsAppCommand_PlanJSONEmitsStructuredPlan proves
// `gormes whatsapp --plan --json` emits a parseable
// `{build, mode, config_path, env_path, session_dir, bridge_log,
// allowed_users, debug, allow_all_users, bridge_command: [...]}` document
// so fleet automation provisioning the WhatsApp Baileys bridge across
// machines can consume the plan without scraping the multi-line preflight
// prose. Build provenance leads — same convention as the rest of the
// `--json` arc. Secrets MUST NOT appear: WhatsApp session credentials live
// inside the session dir, not in the plan output, but a misordered emit
// could leak the dotenv values, so the test asserts the plan stays
// path-only.
func TestWhatsAppCommand_PlanJSONEmitsStructuredPlan(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	stdout, stderr, err := executeWhatsAppCommandForTest(newWhatsAppCommandForTest(), "--plan", "--json", "--mode", "bot", "--allowed-users", "528112345678,528187654321", "--debug")
	if err != nil {
		t.Fatalf("gormes whatsapp --plan --json error = %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}

	var got struct {
		Build struct {
			Version string `json:"version"`
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
	if got.Build.Version != testWhatsAppCommandVersion {
		t.Errorf("build.version = %q, want %q", got.Build.Version, testWhatsAppCommandVersion)
	}
	if got.Mode != "bot" {
		t.Errorf("mode = %q, want bot", got.Mode)
	}
	if got.ConfigPath != config.ConfigPath() {
		t.Errorf("config_path = %q, want %q", got.ConfigPath, config.ConfigPath())
	}
	if got.SessionDir != filepath.Join(home, "whatsapp", "session") {
		t.Errorf("session_dir = %q, want %q", got.SessionDir, filepath.Join(home, "whatsapp", "session"))
	}
	if got.BridgeLog != filepath.Join(home, "whatsapp", "bridge.log") {
		t.Errorf("bridge_log = %q, want %q", got.BridgeLog, filepath.Join(home, "whatsapp", "bridge.log"))
	}
	if got.AllowedUsers != "528112345678,528187654321" {
		t.Errorf("allowed_users = %q, want %q", got.AllowedUsers, "528112345678,528187654321")
	}
	if !got.Debug {
		t.Errorf("debug = false, want true")
	}
	if len(got.BridgeCommand) == 0 {
		t.Errorf("bridge_command must be populated")
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

	stdout, stderr, err := executeWhatsAppCommandForTest(newWhatsAppCommandForTest(), "--mode", "business-api")
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

func executeWhatsAppCommandForTest(cmd *cobra.Command, args ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func exitCodeFromError(err error) int {
	if err == nil {
		return 0
	}
	var coded interface{ ExitCode() int }
	if errors.As(err, &coded) {
		return coded.ExitCode()
	}
	return 1
}

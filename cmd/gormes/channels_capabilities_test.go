package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestChannelsCapabilitiesCommandRendersConfiguredTelegram(t *testing.T) {
	setupChannelsCapabilitiesTestEnv(t)
	writeChannelsCapabilitiesConfig(t, []byte(`
[telegram]
bot_token = "12345:secret-token"
allowed_chat_id = 42
`))

	stdout, stderr, err := executeChannelsCapabilitiesCommand(t, "--channel", "telegram")
	if err != nil {
		t.Fatalf("Execute: %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}
	for _, want := range []string{
		"Telegram (telegram)",
		"Status: configured",
		"Support:",
		"media=partial",
		"Intents:",
		"native_commands",
		"Scopes:",
		"credentials:required",
		"Format limitations:",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "secret-token") {
		t.Fatalf("stdout leaked token:\n%s", stdout)
	}
}

func TestChannelsCapabilitiesCommandJSONOmitsSecrets(t *testing.T) {
	setupChannelsCapabilitiesTestEnv(t)
	writeChannelsCapabilitiesConfig(t, []byte(`
[telegram]
bot_token = "12345:secret-token"
allowed_chat_id = 42
`))

	stdout, stderr, err := executeChannelsCapabilitiesCommand(t, "--channel", "telegram", "--json")
	if err != nil {
		t.Fatalf("Execute: %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}
	if strings.Contains(stdout, "secret-token") {
		t.Fatalf("json leaked token:\n%s", stdout)
	}
	var payload struct {
		Channels []struct {
			Channel      string   `json:"channel"`
			Configured   bool     `json:"configured"`
			ConfigDetail string   `json:"config_detail"`
			Features     []string `json:"features"`
		} `json:"channels"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, stdout)
	}
	if len(payload.Channels) != 1 || payload.Channels[0].Channel != "telegram" {
		t.Fatalf("channels json = %+v, want telegram only", payload.Channels)
	}
	if !payload.Channels[0].Configured || payload.Channels[0].ConfigDetail != "allowed_chat_id=42" {
		t.Fatalf("telegram status = %+v, want configured redacted detail", payload.Channels[0])
	}
}

// TestChannelsCapabilitiesCommandJSONIncludesBuildProvenance proves
// `channels capabilities --json` carries the `build` provenance block
// — same convention as the rest of the --json arc. Operators
// inventorying which channels are configured across hosts can attribute
// each snapshot to the binary that produced it.
func TestChannelsCapabilitiesCommandJSONIncludesBuildProvenance(t *testing.T) {
	setupChannelsCapabilitiesTestEnv(t)
	writeChannelsCapabilitiesConfig(t, []byte(`
[telegram]
bot_token = "12345:secret-token"
allowed_chat_id = 42
`))

	stdout, _, err := executeChannelsCapabilitiesCommand(t, "--channel", "telegram", "--json")
	if err != nil {
		t.Fatalf("channels capabilities --json: %v", err)
	}
	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if got.Build.Version != Version || got.Build.GitCommit == "" {
		t.Fatalf("build provenance missing/wrong: %+v", got.Build)
	}
}

func TestChannelsCapabilitiesCommandTeamsConfiguredStateIsRedacted(t *testing.T) {
	setupChannelsCapabilitiesTestEnv(t)
	writeChannelsCapabilitiesConfig(t, []byte(`
[teams]
enabled = true
client_id = "teams-client"
client_secret = "teams-client-secret"
tenant_id = "teams-tenant"
port = 5001
allowed_users = ["aad-1", "aad-2"]
allow_all_users = true
`))

	stdout, stderr, err := executeChannelsCapabilitiesCommand(t, "--channel", "teams")
	if err != nil {
		t.Fatalf("Execute: %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}
	for _, want := range []string{
		"Microsoft Teams (teams)",
		"Status: configured",
		"configured port=5001 allowed_users=2 allow_all_users=true",
		"credentials:required",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "teams-client-secret") {
		t.Fatalf("stdout leaked Teams secret:\n%s", stdout)
	}

	stdout, stderr, err = executeChannelsCapabilitiesCommand(t, "--channel", "teams", "--json")
	if err != nil {
		t.Fatalf("Execute JSON: %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}
	if strings.Contains(stdout, "teams-client-secret") {
		t.Fatalf("json leaked Teams secret:\n%s", stdout)
	}
	var payload struct {
		Channels []struct {
			Channel      string `json:"channel"`
			Configured   bool   `json:"configured"`
			ConfigDetail string `json:"config_detail"`
		} `json:"channels"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, stdout)
	}
	if len(payload.Channels) != 1 || payload.Channels[0].Channel != "teams" {
		t.Fatalf("channels json = %+v, want teams only", payload.Channels)
	}
	if !payload.Channels[0].Configured {
		t.Fatalf("teams json not configured: %+v", payload.Channels[0])
	}
	if got := payload.Channels[0].ConfigDetail; got != "configured port=5001 allowed_users=2 allow_all_users=true" {
		t.Fatalf("teams config_detail = %q, want redacted configured detail", got)
	}
}

func TestChannelsCapabilitiesCommandUnknownChannelFails(t *testing.T) {
	setupChannelsCapabilitiesTestEnv(t)

	stdout, stderr, err := executeChannelsCapabilitiesCommand(t, "--channel", "missing")
	if err == nil {
		t.Fatalf("Execute returned nil error\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if !strings.Contains(err.Error(), "unknown_channel") {
		t.Fatalf("err = %v, want unknown_channel evidence", err)
	}
}

func setupChannelsCapabilitiesTestEnv(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("GORMES_HOME", filepath.Join(root, "gormes"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("HERMES_HOME", filepath.Join(root, "hermes"))
	t.Setenv("GORMES_TELEGRAM_TOKEN", "")
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_TOKEN", "")
	t.Setenv("TEAMS_CLIENT_ID", "")
	t.Setenv("TEAMS_CLIENT_SECRET", "")
	t.Setenv("TEAMS_TENANT_ID", "")
	t.Setenv("TEAMS_PORT", "")
	t.Setenv("TEAMS_ALLOWED_USERS", "")
	t.Setenv("TEAMS_ALLOW_ALL_USERS", "")
}

func writeChannelsCapabilitiesConfig(t *testing.T, data []byte) {
	t.Helper()
	path := config.ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func executeChannelsCapabilitiesCommand(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	cmd := newRootCommand()
	return executeOneshotFlagCommand(cmd, append([]string{"channels", "capabilities"}, args...)...)
}

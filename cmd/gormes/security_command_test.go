package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestSecurityAuditCommandOutputsJSONAndAppliesSafeFixes(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	t.Setenv("GATEWAY_PROXY_URL", "")
	t.Setenv("GATEWAY_PROXY_KEY", "")
	t.Setenv("GORMES_TELEGRAM_TOKEN", "")
	t.Setenv("GORMES_TELEGRAM_CHAT_ID", "")
	t.Setenv("GORMES_DISCORD_TOKEN", "")
	t.Setenv("GORMES_DISCORD_CHANNEL_ID", "")
	t.Setenv("GORMES_SLACK_ENABLED", "")
	t.Setenv("GORMES_SLACK_BOT_TOKEN", "")
	t.Setenv("GORMES_SLACK_APP_TOKEN", "")
	t.Setenv("GORMES_SLACK_CHANNEL_ID", "")

	const apiSecret = "sk-command-secret"
	const botSecret = "123456:telegram-command-secret"
	configPath := config.ConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`
[hermes]
api_key = "`+apiSecret+`"
endpoint = "http://127.0.0.1:11434"
model = "fixture-model"

[telegram]
bot_token = "`+botSecret+`"
first_run_discovery = true
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.Chmod(configPath, 0o644); err != nil {
		t.Fatalf("chmod config: %v", err)
	}

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "security", "audit", "--deep", "--fix", "--json")
	if err != nil {
		t.Fatalf("security audit: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if strings.Contains(stdout+stderr, apiSecret) || strings.Contains(stdout+stderr, botSecret) {
		t.Fatalf("security audit leaked secret:\nstdout=%s\nstderr=%s", stdout, stderr)
	}

	var result toolspkg.SecurityAuditResult
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("json.Unmarshal: %v\nstdout=%s", err, stdout)
	}
	if !result.OK {
		t.Fatalf("OK = false, want true for warning-only fixed audit: %+v", result)
	}
	if !securityCommandCategoryPresent(result.Categories, toolspkg.SecurityAuditCategoryGatewayAuth) ||
		!securityCommandCategoryPresent(result.Categories, toolspkg.SecurityAuditCategoryStateIntegrity) ||
		!securityCommandCategoryPresent(result.Categories, toolspkg.SecurityAuditCategoryChannelSecurity) ||
		!securityCommandCategoryPresent(result.Categories, toolspkg.SecurityAuditCategoryShellBlocklist) ||
		!securityCommandCategoryPresent(result.Categories, toolspkg.SecurityAuditCategoryFilesystemScoping) ||
		!securityCommandCategoryPresent(result.Categories, toolspkg.SecurityAuditCategoryCredentialRedaction) {
		t.Fatalf("categories = %+v, want all audit categories", result.Categories)
	}
	if !securityCommandFixApplied(result.Fixes, toolspkg.SecurityAuditFixFilePermissions) {
		t.Fatalf("fixes = %+v, missing applied config permission fix", result.Fixes)
	}
	if !securityCommandFixApplied(result.Fixes, toolspkg.SecurityAuditFixGatewayAuthTokenGenerated) {
		t.Fatalf("fixes = %+v, missing gateway auth token generation", result.Fixes)
	}
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("config mode = %o, want 600", got)
	}
	envBody, err := os.ReadFile(config.EnvPath())
	if err != nil {
		t.Fatalf("read env: %v", err)
	}
	if !strings.Contains(string(envBody), "GATEWAY_PROXY_KEY=") {
		t.Fatalf("env file missing generated gateway token:\n%s", envBody)
	}
}

func securityCommandCategoryPresent(categories []toolspkg.SecurityAuditCategoryResult, name string) bool {
	for _, category := range categories {
		if category.Name == name && category.Status != "" {
			return true
		}
	}
	return false
}

func securityCommandFixApplied(fixes []toolspkg.SecurityAuditFix, code string) bool {
	for _, fix := range fixes {
		if fix.Code == code && fix.Applied {
			return true
		}
	}
	return false
}

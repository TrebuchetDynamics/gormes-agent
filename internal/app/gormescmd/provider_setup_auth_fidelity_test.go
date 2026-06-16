package gormescmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

type providerFidelityCheck struct {
	Name    string                 `json:"name"`
	Status  string                 `json:"status"`
	Summary string                 `json:"summary"`
	Items   []providerFidelityItem `json:"items"`
}

type providerFidelityItem struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Note   string `json:"note"`
}

func TestProviderSetupAuthFidelityCodexTruthSurfacesConsistent(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	writeOneshotFlagConfig(t, []byte(`
[hermes]
provider = "openai-codex"
endpoint = "https://chatgpt.com/backend-api/codex"
model = "gpt-5.2"
`))
	if _, err := config.NewCodexOAuthStateStore(config.CodexOAuthStateStoreOptions{}).SaveTokens(config.CodexOAuthTokens{
		AccountID:    "codex-account",
		Label:        "Codex Account",
		AccessToken:  "plain-codex-access",
		RefreshToken: "plain-codex-refresh",
	}); err != nil {
		t.Fatalf("SaveTokens: %v", err)
	}

	authStdout, authStderr, err := executeOneshotFlagCommand(newRootCommandWithRuntime(rootRuntime{}), "auth", "status", config.CodexOAuthProvider, "--json")
	if err != nil {
		t.Fatalf("auth status openai-codex --json: %v\nstdout=%s\nstderr=%s", err, authStdout, authStderr)
	}
	forbidProviderFidelityLeaks(t, authStdout+authStderr)
	var authStatus struct {
		Provider      string `json:"provider"`
		Status        string `json:"status"`
		AuthType      string `json:"auth_type"`
		Authenticated bool   `json:"authenticated"`
		Redacted      bool   `json:"redacted"`
		Credentials   []struct {
			ID              string `json:"id"`
			SecretsRedacted bool   `json:"secrets_redacted"`
		} `json:"credentials"`
	}
	if jsonErr := json.Unmarshal([]byte(authStdout), &authStatus); jsonErr != nil {
		t.Fatalf("auth status stdout must be JSON: %v\n%s", jsonErr, authStdout)
	}
	if authStatus.Provider != config.CodexOAuthProvider || authStatus.Status != "logged_in" || authStatus.AuthType != "oauth_external" || !authStatus.Authenticated || !authStatus.Redacted {
		t.Fatalf("auth status contradiction: %+v", authStatus)
	}
	if len(authStatus.Credentials) != 1 || !authStatus.Credentials[0].SecretsRedacted {
		t.Fatalf("auth status credentials must be present and redacted: %+v", authStatus.Credentials)
	}

	configStdout, configStderr, err := executeOneshotFlagCommand(newRootCommandWithRuntime(rootRuntime{}), "config", "check", "--json")
	if err != nil {
		t.Fatalf("config check --json: %v\nstdout=%s\nstderr=%s", err, configStdout, configStderr)
	}
	forbidProviderFidelityLeaks(t, configStdout+configStderr)
	var configCheck struct {
		OK     bool `json:"ok"`
		Issues []struct {
			Severity string `json:"severity"`
			Field    string `json:"field"`
			Message  string `json:"message"`
		} `json:"issues"`
	}
	if jsonErr := json.Unmarshal([]byte(configStdout), &configCheck); jsonErr != nil {
		t.Fatalf("config check stdout must be JSON: %v\n%s", jsonErr, configStdout)
	}
	if !configCheck.OK {
		t.Fatalf("config check must agree configured provider/model/endpoint are valid: %+v", configCheck)
	}
	if configCheck.Issues == nil {
		t.Fatalf("config check issues must be an array, not null")
	}

	doctorStdout, doctorStderr, err := executeOneshotFlagCommand(newRootCommandWithRuntime(rootRuntime{}), "doctor", "--offline", "--json")
	if err != nil {
		t.Fatalf("doctor --offline --json: %v\nstdout=%s\nstderr=%s", err, doctorStdout, doctorStderr)
	}
	forbidProviderFidelityLeaks(t, doctorStdout+doctorStderr)
	var doctorReport struct {
		Failed bool                    `json:"failed"`
		Checks []providerFidelityCheck `json:"checks"`
	}
	if jsonErr := json.Unmarshal([]byte(doctorStdout), &doctorReport); jsonErr != nil {
		t.Fatalf("doctor stdout must be JSON: %v\n%s", jsonErr, doctorStdout)
	}
	if doctorReport.Failed {
		t.Fatalf("doctor report unexpectedly failed: %+v", doctorReport)
	}
	custom, ok := findProviderFidelityCheck(doctorReport.Checks, "Custom endpoint")
	if !ok {
		t.Fatalf("doctor report missing Custom endpoint check: %+v", doctorReport.Checks)
	}
	if custom.Status != "PASS" || strings.Contains(custom.Summary, "missing=auth") {
		t.Fatalf("Custom endpoint must agree Codex OAuth auth is ready: %+v", custom)
	}
	if _, ok := findProviderFidelityItem(custom.Items, "api_key"); ok {
		t.Fatalf("Codex custom endpoint readiness must not ask for api_key: %+v", custom.Items)
	}
	authItem, ok := findProviderFidelityItem(custom.Items, "auth")
	if !ok || authItem.Status != "PASS" || authItem.Note != "set" {
		t.Fatalf("Custom endpoint auth item must be PASS/set, got item=%+v ok=%t", authItem, ok)
	}

	authProviders, ok := findProviderFidelityCheck(doctorReport.Checks, "Auth Providers")
	if !ok {
		t.Fatalf("doctor report missing Auth Providers check: %+v", doctorReport.Checks)
	}
	codexItem, ok := findProviderFidelityItem(authProviders.Items, "OpenAI Codex")
	if !ok || codexItem.Status != "PASS" || !strings.Contains(codexItem.Note, "status=logged in") || strings.Contains(codexItem.Note, "api_key") {
		t.Fatalf("Auth Providers Codex item must be logged-in OAuth evidence, got item=%+v ok=%t", codexItem, ok)
	}
}

func forbidProviderFidelityLeaks(t *testing.T, text string) {
	t.Helper()
	for _, leak := range []string{"plain-codex-access", "plain-codex-refresh"} {
		if strings.Contains(text, leak) {
			t.Fatalf("provider setup/auth fidelity output leaked %q:\n%s", leak, text)
		}
	}
}

func findProviderFidelityCheck(checks []providerFidelityCheck, name string) (providerFidelityCheck, bool) {
	for _, check := range checks {
		if check.Name == name {
			return check, true
		}
	}
	return providerFidelityCheck{}, false
}

func findProviderFidelityItem(items []providerFidelityItem, name string) (providerFidelityItem, bool) {
	for _, item := range items {
		if item.Name == name {
			return item, true
		}
	}
	return providerFidelityItem{}, false
}

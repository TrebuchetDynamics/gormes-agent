package gormescli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestOnboardExplainsRuntimeSkillsAndLearningState(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_SKILLS_ROOT", "")
	t.Setenv("GORMES_BUNDLED_SKILLS_ROOT", "")

	stdout, stderr, err := executeOnboardCommandWithSeams(t, onboardCommandSeams{})
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	output := stdout
	for _, want := range []string{
		"Gormes onboarding",
		filepath.Join(home, "skills"),
		"Runtime skills",
		"development-skills",
		"manual/prompted",
		"gormes skills list",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("onboard output missing %q:\n%s", want, output)
		}
	}
}

func TestOnboardShowsConfiguredProviderDetails(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_SKILLS_ROOT", "")
	t.Setenv("GORMES_BUNDLED_SKILLS_ROOT", "")
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte("[hermes]\nprovider = 'groq'\nendpoint = 'https://api.groq.com/openai/v1'\nmodel = 'llama-3.3-70b-versatile'\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".env"), []byte("GORMES_API_KEY=sk-onboard-test\n"), 0o600); err != nil {
		t.Fatalf("write env: %v", err)
	}

	stdout, stderr, err := executeOnboardCommandWithSeams(t, onboardCommandSeams{})
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	output := stdout
	for _, want := range []string{
		"Provider: groq",
		"Endpoint: https://api.groq.com/openai/v1",
		"Model: llama-3.3-70b-versatile",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("onboard output missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "No provider configured yet") {
		t.Fatalf("onboard treated configured provider as missing:\n%s", output)
	}
}

func TestOnboardShowsProviderDetailsWhenCodexAuthMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_SKILLS_ROOT", "")
	t.Setenv("GORMES_BUNDLED_SKILLS_ROOT", "")
	t.Setenv("GORMES_API_KEY", "")
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte("[hermes]\nprovider = 'openai-codex'\nendpoint = 'https://chatgpt.com/backend-api/codex'\nmodel = 'gpt-5.2'\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	stdout, stderr, err := executeOnboardCommandWithSeams(t, onboardCommandSeams{})
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	output := stdout
	for _, want := range []string{
		"Provider: openai-codex",
		"Endpoint: https://chatgpt.com/backend-api/codex",
		"Model: gpt-5.2",
		"Auth: missing",
		"gormes auth add openai-codex",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("onboard output missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "No provider configured yet") {
		t.Fatalf("onboard treated selected Codex provider as missing:\n%s", output)
	}
}

func TestOnboardShowsCodexCredentialPoolAuthConfigured(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_SKILLS_ROOT", "")
	t.Setenv("GORMES_BUNDLED_SKILLS_ROOT", "")
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte("[hermes]\nprovider = 'openai-codex'\nendpoint = 'https://chatgpt.com/backend-api/codex'\nmodel = 'gpt-5.2'\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := config.SaveCredentialPoolEntries(config.CredentialPoolOptions{Provider: config.CodexOAuthProvider}, []config.PooledCredential{{
		ID:           "codex-import",
		Label:        "Imported Codex CLI",
		AuthType:     config.CredentialAuthOAuth,
		Source:       config.CodexOAuthSourceCodexCLIImport,
		AccessToken:  "codex-access-secret",
		RefreshToken: "codex-refresh-secret",
		LastStatus:   config.CredentialStatusOK,
	}}); err != nil {
		t.Fatalf("SaveCredentialPoolEntries: %v", err)
	}

	stdout, stderr, err := executeOnboardCommandWithSeams(t, onboardCommandSeams{})
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	output := stdout
	for _, want := range []string{
		"Provider: openai-codex",
		"Auth: configured",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("onboard output missing %q:\n%s", want, output)
		}
	}
	for _, forbidden := range []string{"Auth: missing", "codex-access-secret", "codex-refresh-secret"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("onboard output contained forbidden %q:\n%s", forbidden, output)
		}
	}
}

func TestOnboardWizardShowsCodexCredentialPoolAuthConfigured(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_SKILLS_ROOT", "")
	t.Setenv("GORMES_BUNDLED_SKILLS_ROOT", "")
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte("[hermes]\nprovider = 'openai-codex'\nendpoint = 'https://chatgpt.com/backend-api/codex'\nmodel = 'gpt-5.2'\n[telegram]\nallowed_user_ids = [6586915095]\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".env"), []byte("GORMES_TELEGRAM_TOKEN=telegram-secret\n"), 0o600); err != nil {
		t.Fatalf("write env: %v", err)
	}
	if err := config.SaveCredentialPoolEntries(config.CredentialPoolOptions{Provider: config.CodexOAuthProvider}, []config.PooledCredential{{
		ID:           "codex-import",
		Label:        "Imported Codex CLI",
		AuthType:     config.CredentialAuthOAuth,
		Source:       config.CodexOAuthSourceCodexCLIImport,
		AccessToken:  "codex-access-secret",
		RefreshToken: "codex-refresh-secret",
		LastStatus:   config.CredentialStatusOK,
	}}); err != nil {
		t.Fatalf("SaveCredentialPoolEntries: %v", err)
	}

	stdout, stderr, err := executeOnboardCommandWithSeams(t, onboardCommandSeams{}, "--wizard", "--non-interactive")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	output := stdout
	for _, want := range []string{
		"Provider: configured",
		"openai-codex",
		"Auth: configured",
		"Provider credential present",
		"Gateway: configured",
		"telegram",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("onboard wizard output missing %q:\n%s", want, output)
		}
	}
	for _, forbidden := range []string{"Auth: missing", "codex-access-secret", "codex-refresh-secret"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("onboard wizard output contained forbidden %q:\n%s", forbidden, output)
		}
	}
}

func TestOnboardWizardNonInteractiveShowsOrderedPlanAndSkipWarnings(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_SKILLS_ROOT", "")
	t.Setenv("GORMES_BUNDLED_SKILLS_ROOT", "")

	stdout, stderr, err := executeOnboardCommandWithSeams(t, onboardCommandSeams{}, "--wizard", "--non-interactive")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	output := stdout
	for _, want := range []string{
		"Gormes first-run wizard",
		"Mode: non-interactive plan",
		"1. Model:",
		"2. Provider:",
		"3. Auth:",
		"4. Gateway:",
		"5. Browser/CDP:",
		"6. Skills:",
		"7. Dashboard:",
		"Skip warning:",
		"gormes setup model",
		"gormes setup provider",
		"gormes auth add",
		"gormes setup gateway",
		"gormes doctor --offline",
		"gormes skills list",
		"gormes dashboard",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("onboard wizard output missing %q:\n%s", want, output)
		}
	}
}

func TestOnboardWizardPrefillsConfiguredProviderAndAuth(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_SKILLS_ROOT", "")
	t.Setenv("GORMES_BUNDLED_SKILLS_ROOT", "")
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte("[hermes]\nprovider = 'groq'\nendpoint = 'https://api.groq.com/openai/v1'\nmodel = 'llama-3.3-70b-versatile'\n[browser]\ncdp_url = 'http://127.0.0.1:9222'\n[telegram]\nbot_token = 'telegram-token'\nallowed_chat_id = 42\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".env"), []byte("GORMES_API_KEY=sk-onboard-test\n"), 0o600); err != nil {
		t.Fatalf("write env: %v", err)
	}

	stdout, stderr, err := executeOnboardCommandWithSeams(t, onboardCommandSeams{}, "--wizard", "--non-interactive")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	output := stdout
	for _, want := range []string{
		"Model: configured",
		"llama-3.3-70b-versatile",
		"Provider: configured",
		"groq",
		"https://api.groq.com/openai/v1",
		"Auth: configured",
		"credential present",
		"Gateway: configured",
		"telegram",
		"Browser/CDP: configured",
		"http://127.0.0.1:9222",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("onboard wizard output missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "No provider configured yet") {
		t.Fatalf("wizard reused status missing-provider copy:\n%s", output)
	}
}

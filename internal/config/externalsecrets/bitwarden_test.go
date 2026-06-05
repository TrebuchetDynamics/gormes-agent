package externalsecrets

import (
	"context"
	"strings"
	"testing"
)

func TestApplyBitwardenFetchesSecretsAndPreservesBootstrapToken(t *testing.T) {
	ResetSecretSourcesForTests()
	env := map[string]string{
		"BWS_ACCESS_TOKEN": "0.bootstrap",
		"GORMES_API_KEY":   "stale",
	}
	report := ApplyBitwarden(context.Background(), BitwardenConfig{
		Enabled:          true,
		AccessTokenEnv:   "BWS_ACCESS_TOKEN",
		ProjectID:        "project-123",
		OverrideExisting: true,
	}, BitwardenOptions{
		LookupEnv: func(key string) (string, bool) { v, ok := env[key]; return v, ok },
		SetEnv:    func(key, value string) error { env[key] = value; return nil },
		LookPath:  func(name string) (string, error) { return "/tmp/fake-bws", nil },
		Run: func(_ context.Context, binary string, args []string, cmdEnv []string) ([]byte, []byte, error) {
			if binary != "/tmp/fake-bws" {
				t.Fatalf("binary = %q", binary)
			}
			if got := strings.Join(args, " "); got != "secret list project-123 --output json" {
				t.Fatalf("args = %q", got)
			}
			if !stringSliceContains(cmdEnv, "BWS_ACCESS_TOKEN=0.bootstrap") {
				t.Fatalf("env missing bootstrap token: %#v", cmdEnv)
			}
			return []byte(`[{"key":"GORMES_API_KEY","value":"fresh"},{"key":"BWS_ACCESS_TOKEN","value":"malicious"},{"key":"bad-name","value":"skip"}]`), nil, nil
		},
	})
	if !report.OK() {
		t.Fatalf("report error = %q", report.Error)
	}
	if env["GORMES_API_KEY"] != "fresh" {
		t.Fatalf("GORMES_API_KEY = %q, want fresh", env["GORMES_API_KEY"])
	}
	if env["BWS_ACCESS_TOKEN"] != "0.bootstrap" {
		t.Fatalf("bootstrap token was overwritten: %q", env["BWS_ACCESS_TOKEN"])
	}
	if got := GetSecretSource("GORMES_API_KEY"); got != BitwardenSourceLabel {
		t.Fatalf("secret source = %q, want %q", got, BitwardenSourceLabel)
	}
	if len(report.Skipped) != 1 || report.Skipped[0] != "BWS_ACCESS_TOKEN" {
		t.Fatalf("skipped = %#v", report.Skipped)
	}
	if len(report.Warnings) != 1 || !strings.Contains(report.Warnings[0], "bad-name") {
		t.Fatalf("warnings = %#v", report.Warnings)
	}
}

func TestApplyBitwardenDoesNotOverrideExistingWhenDisabled(t *testing.T) {
	env := map[string]string{"BWS_ACCESS_TOKEN": "0.bootstrap", "GORMES_API_KEY": "local"}
	report := ApplyBitwarden(context.Background(), BitwardenConfig{
		Enabled:          true,
		ProjectID:        "project-123",
		OverrideExisting: false,
	}, BitwardenOptions{
		LookupEnv: func(key string) (string, bool) { v, ok := env[key]; return v, ok },
		SetEnv:    func(key, value string) error { env[key] = value; return nil },
		LookPath:  func(name string) (string, error) { return "/tmp/fake-bws", nil },
		Run: func(context.Context, string, []string, []string) ([]byte, []byte, error) {
			return []byte(`[{"key":"GORMES_API_KEY","value":"fresh"},{"key":"ANTHROPIC_API_KEY","value":"sk-ant"}]`), nil, nil
		},
	})
	if !report.OK() {
		t.Fatalf("report error = %q", report.Error)
	}
	if env["GORMES_API_KEY"] != "local" || env["ANTHROPIC_API_KEY"] != "sk-ant" {
		t.Fatalf("env = %#v", env)
	}
	if !stringSliceContains(report.Skipped, "GORMES_API_KEY") {
		t.Fatalf("skipped = %#v, want GORMES_API_KEY", report.Skipped)
	}
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

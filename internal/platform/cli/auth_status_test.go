package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestAuthStatusCredentialPoolLoggedInRedactsSecrets(t *testing.T) {
	var out bytes.Buffer
	status, err := RenderAuthStatus(context.Background(), &out, "openrouter", AuthStatusOptions{
		CredentialLoader: func(provider string) (config.CredentialPoolStatus, config.CredentialPoolEvidence, error) {
			if provider != "openrouter" {
				t.Fatalf("provider = %q, want openrouter", provider)
			}
			return config.CredentialPoolStatus{
				Provider: provider,
				Count:    1,
				Redacted: true,
				Entries:  []config.RedactedCredentialStatus{{ID: "openrouter-manual-1", Label: "personal", AuthType: config.CredentialAuthAPIKey, Source: "manual", LastStatus: config.CredentialStatusExhausted, LastErrorReason: "rate_limited", SecretsRedacted: true}},
			}, config.CredentialPoolEvidence{Code: config.CredentialPoolEvidenceLoaded, Provider: provider, Redacted: true}, nil
		},
	})
	if err != nil {
		t.Fatalf("RenderAuthStatus: %v", err)
	}
	if !status.Authenticated || status.Provider != "openrouter" || status.AuthType != "api_key" {
		t.Fatalf("status = %#v", status)
	}
	for _, want := range []string{"auth_status provider=openrouter status=logged_in", "credentials=1", "rate_limited", "redacted=true"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
	for _, leak := range []string{"plain-existing-token", "plain-refresh-token"} {
		if strings.Contains(out.String(), leak) {
			t.Fatalf("output leaked %q:\n%s", leak, out.String())
		}
	}
}

func TestAuthStatusCredentialPoolLoggedOutReason(t *testing.T) {
	var out bytes.Buffer
	status, err := RenderAuthStatus(context.Background(), &out, "openrouter", AuthStatusOptions{
		CredentialLoader: func(provider string) (config.CredentialPoolStatus, config.CredentialPoolEvidence, error) {
			return config.CredentialPoolStatus{Provider: provider, Count: 0, Redacted: true}, config.CredentialPoolEvidence{Code: config.CredentialPoolEvidenceEmpty, Provider: provider, Redacted: true}, nil
		},
	})
	if err != nil {
		t.Fatalf("RenderAuthStatus: %v", err)
	}
	if status.Status != AuthStatusLoggedOut || status.Reason != config.CredentialPoolEvidenceEmpty {
		t.Fatalf("status = %#v", status)
	}
	if !strings.Contains(out.String(), "status=logged_out") || !strings.Contains(out.String(), "reason=credential_pool_empty") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestAuthStatusCodexUsesInjectedOAuthReadModel(t *testing.T) {
	var out bytes.Buffer
	status, err := RenderAuthStatus(context.Background(), &out, config.CodexOAuthProvider, AuthStatusOptions{
		CodexStatus: func() (config.CodexOAuthAuthStatus, error) {
			return config.CodexOAuthAuthStatus{Code: config.CodexOAuthStatusAuthorized, Provider: config.CodexOAuthProvider, AccountID: "codex-device-1", Label: "codex", Authenticated: true, Source: config.CodexOAuthSourceDeviceCode, Redacted: true}, nil
		},
	})
	if err != nil {
		t.Fatalf("RenderAuthStatus: %v", err)
	}
	if !status.Authenticated || status.AuthType != "oauth_external" {
		t.Fatalf("status = %#v", status)
	}
	for _, want := range []string{"provider=openai-codex", "auth_type=oauth_external", "codex-device-1", "device-code"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
}

func TestAuthStatusAnthropicUsesInjectedReadModel(t *testing.T) {
	status, err := ResolveAuthStatus(context.Background(), config.AnthropicProvider, AuthStatusOptions{
		AnthropicStatus: func(context.Context) (config.AnthropicAuthStatus, error) {
			return config.AnthropicAuthStatus{Code: config.AnthropicAuthStatusReloginRequired, Provider: config.AnthropicProvider, ReloginRequired: true, Evidence: config.AnthropicOAuthEvidenceStaleOAuth, Redacted: true}, nil
		},
	})
	if err != nil {
		t.Fatalf("ResolveAuthStatus: %v", err)
	}
	if status.Status != AuthStatusLoggedOut || status.Reason != config.AnthropicAuthStatusReloginRequired || status.AuthType != config.CredentialAuthOAuth {
		t.Fatalf("status = %#v", status)
	}
}

func TestAuthStatusRejectsUnknownProvider(t *testing.T) {
	_, err := ResolveAuthStatus(context.Background(), "not-a-provider", AuthStatusOptions{})
	if err == nil || !strings.Contains(err.Error(), "unknown provider") {
		t.Fatalf("err = %v, want unknown provider", err)
	}
}

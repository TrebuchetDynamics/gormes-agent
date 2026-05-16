package doctor

import (
	"strings"
	"testing"
)

func TestCheckAuthProvidersEnumeratesProviderStatuses(t *testing.T) {
	got := CheckAuthProviders([]AuthProviderStatus{
		{
			Name:            "OpenAI Codex",
			Provider:        "openai-codex",
			AuthType:        "oauth_external",
			Status:          AuthProviderLoggedIn,
			Authenticated:   true,
			Reason:          "authorized",
			CredentialCount: 1,
			Redacted:        true,
		},
		{
			Name:     "Nous Portal",
			Provider: "nous",
			AuthType: "oauth_device_code",
			Status:   AuthProviderLoggedOut,
			Reason:   "credential_pool_empty",
			Redacted: true,
		},
		{
			Name:     "Custom endpoint",
			Provider: "custom",
			AuthType: "api_key",
			Status:   AuthProviderSkipped,
			Reason:   "not configured",
			Redacted: true,
		},
	})

	if got.Name != "Auth Providers" {
		t.Fatalf("Name = %q, want Auth Providers", got.Name)
	}
	if got.Status != StatusWarn {
		t.Fatalf("Status = %v, want WARN because Nous is logged out: %+v", got.Status, got)
	}
	items := authProviderItemsByName(got)
	if it := items["OpenAI Codex"]; it.Status != StatusPass || !strings.Contains(it.Note, "logged in") || !strings.Contains(it.Note, "credentials=1") {
		t.Fatalf("OpenAI Codex item = %+v, want logged-in credential-pool PASS", it)
	}
	if it := items["Nous Portal"]; it.Status != StatusWarn || !strings.Contains(it.Note, "not logged in") || !strings.Contains(it.Note, "credential_pool_empty") {
		t.Fatalf("Nous item = %+v, want not-logged-in WARN", it)
	}
	if it := items["Custom endpoint"]; it.Status != StatusSkip || !strings.Contains(it.Note, "not configured") {
		t.Fatalf("Custom endpoint item = %+v, want SKIP not configured", it)
	}
}

func TestCheckAuthProvidersRedactsTokenLikeReasons(t *testing.T) {
	got := CheckAuthProviders([]AuthProviderStatus{{
		Name:     "OpenAI Codex",
		Provider: "openai-codex",
		AuthType: "oauth_external",
		Status:   AuthProviderError,
		Reason:   "access_token=plain-codex-access refresh_token=plain-codex-refresh",
		Redacted: true,
	}})

	out := got.Format()
	for _, leak := range []string{"plain-codex-access", "plain-codex-refresh", "access_token=", "refresh_token="} {
		if strings.Contains(out, leak) {
			t.Fatalf("Auth Providers leaked %q:\n%s", leak, out)
		}
	}
	if !strings.Contains(out, "[redacted]") {
		t.Fatalf("Auth Providers should keep redaction evidence:\n%s", out)
	}
}

func authProviderItemsByName(r CheckResult) map[string]ItemInfo {
	out := map[string]ItemInfo{}
	for _, it := range r.Items {
		out[it.Name] = it
	}
	return out
}

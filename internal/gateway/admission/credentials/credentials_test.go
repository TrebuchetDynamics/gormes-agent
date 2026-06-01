package credentials

import "testing"

func TestGatewayWeakCredentialGuardDisablesEnabledPlaceholderPlatforms(t *testing.T) {
	report := CheckWeakCredentialPlatforms([]CredentialGuardPlatform{
		{
			Name:    "telegram",
			Enabled: true,
			Credentials: []CredentialGuardValue{{
				Field: "bot_token",
				Value: " *** ",
			}},
		},
		{
			Name:    "discord",
			Enabled: true,
			Credentials: []CredentialGuardValue{{
				Field: "token",
				Value: "changeme",
			}},
		},
		{
			Name:    "slack",
			Enabled: true,
			Credentials: []CredentialGuardValue{{
				Field: "bot_token",
				Value: "your_api_key",
			}},
		},
		{
			Name:    "matrix",
			Enabled: true,
			Credentials: []CredentialGuardValue{{
				Field: "access_token",
				Value: " placeholder ",
			}},
		},
	})

	if len(report.DisabledPlatforms) != 4 {
		t.Fatalf("DisabledPlatforms = %#v, want 4 platforms", report.DisabledPlatforms)
	}
	for _, evidence := range report.Evidence {
		if evidence.Code != "gateway_weak_credential_disabled" {
			t.Fatalf("Code = %q, want gateway_weak_credential_disabled", evidence.Code)
		}
		if evidence.Secret != "" {
			t.Fatalf("evidence leaked secret: %#v", evidence)
		}
	}
}

func TestGatewayWeakCredentialGuardIgnoresDisabledOrEmptyTokens(t *testing.T) {
	report := CheckWeakCredentialPlatforms([]CredentialGuardPlatform{
		{
			Name:    "disabled",
			Enabled: false,
			Credentials: []CredentialGuardValue{{
				Field: "token",
				Value: "placeholder",
			}},
		},
		{
			Name:    "empty",
			Enabled: true,
			Credentials: []CredentialGuardValue{{
				Field: "token",
				Value: " ",
			}},
		},
	})
	if len(report.DisabledPlatforms) != 0 || len(report.Evidence) != 0 {
		t.Fatalf("report = %#v, want no disabled platforms/evidence", report)
	}
}

package admission

import "testing"

func TestFacadePreservesAdmissionContracts(t *testing.T) {
	if got := CheckStartupAllowlist(StartupAdmissionInput{}); len(got) != 1 || got[0].Code != "gateway_allowlist_missing" {
		t.Fatalf("CheckStartupAllowlist facade = %#v, want missing-allowlist evidence", got)
	}

	credentials := []CredentialGuardPlatform{{
		Name:    "telegram",
		Enabled: true,
		Credentials: []CredentialGuardValue{{
			Field: "bot_token",
			Value: "placeholder",
		}},
	}}
	if got := CheckWeakCredentialPlatforms(credentials); len(got.DisabledPlatforms) != 1 || got.Evidence[0].Secret != "" {
		t.Fatalf("CheckWeakCredentialPlatforms facade = %#v, want one redacted disabled platform", got)
	}
}

func TestFacadePreservesWhitelistContracts(t *testing.T) {
	wc := ParseWhitelistConfig([]string{" chat-a ", "chat-a", "chat-b"})
	if !wc.Enabled || len(wc.IDs) != 2 || !wc.IsAllowed("chat-a") || wc.IsAllowed("chat-c") {
		t.Fatalf("ParseWhitelistConfig facade = %#v, want compact enabled whitelist", wc)
	}
}

package gateway

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestProfileChannelReadinessAllowsSameChannelDifferentProfileCredentialsAndAllowLists(t *testing.T) {
	cfg := config.Config{
		Profiles: map[string]config.ProfileCfg{
			"main": {
				Enabled: true,
				Channels: map[string]config.ProfileChannelCfg{
					"whatsapp": {
						Enabled:        true,
						Credential:     "main-whatsapp",
						AllowedChats:   []string{"12025550123@s.whatsapp.net"},
						AllowedUsers:   []string{"6586915095"},
						RequireMention: true,
						ToolProgress:   "compact",
					},
				},
			},
			"sales": {
				Enabled: true,
				Channels: map[string]config.ProfileChannelCfg{
					"whatsapp": {
						Enabled:      true,
						Credential:   "sales-whatsapp",
						AllowedChats: []string{"12025550999-123@g.us"},
						AllowedUsers: []string{"7770001111"},
					},
				},
			},
		},
		Credentials: map[string]config.CredentialCfg{
			"main-whatsapp": {
				Kind:         "channel",
				Channel:      "whatsapp",
				OwnerProfile: "main",
				SecretRef: &config.SecretRef{
					Source: config.SecretRefSourceEnv,
					ID:     "GORMES_MAIN_WHATSAPP_TOKEN",
				},
			},
			"sales-whatsapp": {
				Kind:         "channel",
				Channel:      "whatsapp",
				OwnerProfile: "sales",
				SecretRef: &config.SecretRef{
					Source: config.SecretRefSourceEnv,
					ID:     "GORMES_SALES_WHATSAPP_TOKEN",
				},
			},
		},
	}

	report := BuildProfileChannelReadiness(cfg)
	mainBinding := findProfileChannelBinding(t, report, "main", "whatsapp")
	salesBinding := findProfileChannelBinding(t, report, "sales", "whatsapp")
	if !mainBinding.Ready || !salesBinding.Ready {
		t.Fatalf("readiness = main:%+v sales:%+v, want both ready", mainBinding, salesBinding)
	}
	if mainBinding.CredentialID == salesBinding.CredentialID {
		t.Fatalf("credential IDs unexpectedly shared: main=%q sales=%q", mainBinding.CredentialID, salesBinding.CredentialID)
	}
	if mainBinding.AllowedChatCount != 1 || salesBinding.AllowedChatCount != 1 {
		t.Fatalf("allowed chat counts = main:%d sales:%d, want 1 each", mainBinding.AllowedChatCount, salesBinding.AllowedChatCount)
	}
	if mainBinding.AllowedChatScopeHash == "" || salesBinding.AllowedChatScopeHash == "" || mainBinding.AllowedChatScopeHash == salesBinding.AllowedChatScopeHash {
		t.Fatalf("allowed chat scope hashes = main:%q sales:%q, want non-empty distinct redacted scopes", mainBinding.AllowedChatScopeHash, salesBinding.AllowedChatScopeHash)
	}
	if mainBinding.AllowedUserScopeHash == "" || salesBinding.AllowedUserScopeHash == "" || mainBinding.AllowedUserScopeHash == salesBinding.AllowedUserScopeHash {
		t.Fatalf("allowed user scope hashes = main:%q sales:%q, want non-empty distinct redacted scopes", mainBinding.AllowedUserScopeHash, salesBinding.AllowedUserScopeHash)
	}
	if !mainBinding.SecretRefConfigured || mainBinding.SecretRefSource != "env" {
		t.Fatalf("main secret ref readiness = configured:%v source:%q, want configured env", mainBinding.SecretRefConfigured, mainBinding.SecretRefSource)
	}

	body, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal readiness report: %v", err)
	}
	for _, leaked := range []string{
		"12025550123", "12025550999", "6586915095", "7770001111",
		"GORMES_MAIN_WHATSAPP_TOKEN", "GORMES_SALES_WHATSAPP_TOKEN",
	} {
		if strings.Contains(string(body), leaked) {
			t.Fatalf("readiness report leaked sensitive value %q:\n%s", leaked, body)
		}
	}
}

func TestProfileChannelReadinessWhatsAppAllowListsCanonicalizeJIDCase(t *testing.T) {
	cfg := config.Config{
		Profiles: map[string]config.ProfileCfg{
			"main": {
				Enabled: true,
				Channels: map[string]config.ProfileChannelCfg{
					"whatsapp": {
						Enabled:    true,
						Credential: "main-whatsapp",
						AllowedChats: []string{
							"12025550123@S.WHATSAPP.NET",
							"12025550123@s.whatsapp.net",
							"12025550999-123@G.US",
							"12025550999-123@g.us",
						},
						AllowedUsers: []string{"6586915095", " 6586915095 "},
					},
				},
			},
		},
		Credentials: map[string]config.CredentialCfg{
			"main-whatsapp": channelCredential("whatsapp", "main", "GORMES_MAIN_WHATSAPP_TOKEN"),
		},
	}

	report := BuildProfileChannelReadiness(cfg)
	binding := findProfileChannelBinding(t, report, "main", "whatsapp")
	if !binding.Ready {
		t.Fatalf("binding Ready = false, want ready after canonicalizing duplicated WhatsApp JID case: %+v", binding)
	}
	if binding.AllowedChatCount != 2 || binding.AllowedDirectChatCount != 1 || binding.AllowedGroupChatCount != 1 {
		t.Fatalf("chat counts = total:%d direct:%d group:%d, want 2/1/1 after case canonicalization", binding.AllowedChatCount, binding.AllowedDirectChatCount, binding.AllowedGroupChatCount)
	}
	if binding.AllowedUserCount != 1 {
		t.Fatalf("AllowedUserCount = %d, want duplicate trimmed user collapsed", binding.AllowedUserCount)
	}

	body, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal readiness report: %v", err)
	}
	for _, leaked := range []string{"12025550123", "12025550999", "6586915095", "GORMES_MAIN_WHATSAPP_TOKEN"} {
		if strings.Contains(string(body), leaked) {
			t.Fatalf("readiness report leaked sensitive value %q:\n%s", leaked, body)
		}
	}
}

func TestProfileChannelReadinessMissingCredentialSkipsOnlyThatBinding(t *testing.T) {
	cfg := config.Config{
		Profiles: map[string]config.ProfileCfg{
			"main": {
				Enabled: true,
				Channels: map[string]config.ProfileChannelCfg{
					"whatsapp": {Enabled: true, Credential: "missing-whatsapp"},
				},
			},
			"ops": {
				Enabled: true,
				Channels: map[string]config.ProfileChannelCfg{
					"telegram": {Enabled: true, Credential: "ops-telegram", AllowedUsers: []string{"42"}},
				},
			},
		},
		Credentials: map[string]config.CredentialCfg{
			"ops-telegram": {
				Kind:         "channel",
				Channel:      "telegram",
				OwnerProfile: "ops",
				SecretRef: &config.SecretRef{
					Source: config.SecretRefSourceEnv,
					ID:     "GORMES_OPS_TELEGRAM_TOKEN",
				},
			},
		},
	}

	report := BuildProfileChannelReadiness(cfg)
	missing := findProfileChannelBinding(t, report, "main", "whatsapp")
	ready := findProfileChannelBinding(t, report, "ops", "telegram")
	if missing.Ready {
		t.Fatalf("missing binding Ready = true, want skipped/degraded: %+v", missing)
	}
	if !hasProfileChannelEvidence(missing.Evidence, "channel_credential_missing") {
		t.Fatalf("missing binding evidence = %+v, want channel_credential_missing", missing.Evidence)
	}
	if !ready.Ready {
		t.Fatalf("unrelated binding Ready = false, want available despite missing WhatsApp credential: %+v", ready)
	}
	if len(report.Bindings) != 2 {
		t.Fatalf("bindings = %d, want both degraded and ready bindings present", len(report.Bindings))
	}
}

func TestProfileChannelReadinessDuplicateTokenHashConflictsAcrossProfiles(t *testing.T) {
	const rawSharedToken = "123456:shared-whatsapp-token-that-must-not-leak"
	sharedHash := TokenCredentialHash(rawSharedToken)
	cfg := config.Config{
		Profiles: map[string]config.ProfileCfg{
			"main": {
				Enabled: true,
				Channels: map[string]config.ProfileChannelCfg{
					"whatsapp": {Enabled: true, Credential: "main-whatsapp"},
				},
			},
			"sales": {
				Enabled: true,
				Channels: map[string]config.ProfileChannelCfg{
					"whatsapp": {Enabled: true, Credential: "sales-whatsapp"},
				},
			},
			"ops": {
				Enabled: true,
				Channels: map[string]config.ProfileChannelCfg{
					"telegram": {Enabled: true, Credential: "ops-telegram"},
				},
			},
		},
		Credentials: map[string]config.CredentialCfg{
			"main-whatsapp":  channelCredential("whatsapp", "main", "GORMES_MAIN_WHATSAPP_TOKEN"),
			"sales-whatsapp": channelCredential("whatsapp", "sales", "GORMES_SALES_WHATSAPP_TOKEN"),
			"ops-telegram":   channelCredential("telegram", "ops", "GORMES_OPS_TELEGRAM_TOKEN"),
		},
	}

	report := BuildProfileChannelReadinessWithOptions(cfg, ProfileChannelReadinessOptions{
		CredentialHashes: map[string]string{
			"main-whatsapp":  sharedHash,
			"sales-whatsapp": sharedHash,
			"ops-telegram":   TokenCredentialHash("other-token"),
		},
	})
	mainBinding := findProfileChannelBinding(t, report, "main", "whatsapp")
	salesBinding := findProfileChannelBinding(t, report, "sales", "whatsapp")
	opsBinding := findProfileChannelBinding(t, report, "ops", "telegram")
	for _, binding := range []ProfileChannelBindingReadiness{mainBinding, salesBinding} {
		if binding.Ready {
			t.Fatalf("%s binding Ready = true, want token ownership conflict evidence: %+v", binding.ProfileID, binding)
		}
		if binding.CredentialHash != sharedHash {
			t.Fatalf("%s credential hash = %q, want shared hash %q", binding.ProfileID, binding.CredentialHash, sharedHash)
		}
		if !hasProfileChannelEvidence(binding.Evidence, "channel_token_hash_conflict") {
			t.Fatalf("%s evidence = %+v, want channel_token_hash_conflict", binding.ProfileID, binding.Evidence)
		}
	}
	if !opsBinding.Ready {
		t.Fatalf("unrelated telegram binding Ready = false, want ready: %+v", opsBinding)
	}

	body, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal readiness report: %v", err)
	}
	for _, leaked := range []string{
		rawSharedToken, "shared-whatsapp-token", "GORMES_MAIN_WHATSAPP_TOKEN", "GORMES_SALES_WHATSAPP_TOKEN",
	} {
		if strings.Contains(string(body), leaked) {
			t.Fatalf("readiness report leaked sensitive value %q:\n%s", leaked, body)
		}
	}
}

func findProfileChannelBinding(t *testing.T, report ProfileChannelReadinessReport, profileID, channel string) ProfileChannelBindingReadiness {
	t.Helper()
	for _, binding := range report.Bindings {
		if binding.ProfileID == profileID && binding.Channel == channel {
			return binding
		}
	}
	t.Fatalf("missing profile channel binding profile=%q channel=%q in %+v", profileID, channel, report.Bindings)
	return ProfileChannelBindingReadiness{}
}

func hasProfileChannelEvidence(items []ProfileChannelReadinessEvidence, code string) bool {
	for _, item := range items {
		if item.Code == code {
			return true
		}
	}
	return false
}

func channelCredential(channel, ownerProfile, envID string) config.CredentialCfg {
	return config.CredentialCfg{
		Kind:         "channel",
		Channel:      channel,
		OwnerProfile: ownerProfile,
		SecretRef: &config.SecretRef{
			Source: config.SecretRefSourceEnv,
			ID:     envID,
		},
	}
}

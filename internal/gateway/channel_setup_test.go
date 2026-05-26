package gateway

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestChannelSetupTelegramStatusAndRedaction(t *testing.T) {
	const token = "123456:secret-token-that-must-not-leak"
	plan := BuildChannelSetupPlan(config.Config{
		Telegram: config.TelegramCfg{BotToken: token},
	})
	telegram := findChannelSetupEntry(t, plan, "telegram")
	if telegram.Status != ChannelSetupStatusPartial {
		t.Fatalf("telegram status = %q, want partial for token-only config", telegram.Status)
	}
	if !strings.Contains(strings.Join(telegram.Warnings, "\n"), "access policy") {
		t.Fatalf("telegram warnings = %v, want access-policy warning", telegram.Warnings)
	}
	rendered := strings.Join(append(telegram.CurrentValues, telegram.PlannedWrites...), "\n")
	if strings.Contains(rendered, token) {
		t.Fatalf("channel setup plan leaked token in current/planned values: %s", rendered)
	}
	if !strings.Contains(rendered, "telegram.bot_token=[REDACTED]") {
		t.Fatalf("channel setup plan missing redacted token evidence: %s", rendered)
	}

	plan = BuildChannelSetupPlan(config.Config{
		Telegram: config.TelegramCfg{
			BotToken:       token,
			AllowedUserIDs: []int64{6586915095},
			HomeChannel: config.TelegramHomeChannelCfg{
				ChatID:   "-1001234567890",
				ThreadID: "42",
			},
		},
	})
	telegram = findChannelSetupEntry(t, plan, "telegram")
	if telegram.Status != ChannelSetupStatusConfigured {
		t.Fatalf("telegram status = %q, want configured with token, allowlist, and home channel", telegram.Status)
	}
	for _, want := range []string{
		"telegram.allowed_user_ids=1",
		"telegram.home_channel.chat_id=-1001234567890",
		"telegram.home_channel.thread_id=42",
	} {
		if !strings.Contains(strings.Join(telegram.CurrentValues, "\n"), want) {
			t.Fatalf("telegram current values missing %q: %v", want, telegram.CurrentValues)
		}
	}
}

func TestChannelSetupPlanCoreChannelsAdvertiseProfileScopedWrites(t *testing.T) {
	plan := BuildChannelSetupPlan(config.Config{})
	checks := map[string][]string{
		"telegram": {"profiles.<id>.channels.telegram.credential", "credentials.<id>.secret_ref", "profile-scoped .env"},
		"discord":  {"profiles.<id>.channels.discord.credential", "credentials.<id>.secret_ref", "profile-scoped .env"},
		"slack":    {"profiles.<id>.channels.slack.credential", "credentials.<id>.secret_ref", "profile-scoped .env"},
		"navivox":  {"profiles.<id>.channels.navivox.credential", "credentials.<id>.secret_ref", "profile-scoped .env"},
	}
	for channel, wants := range checks {
		entry := findChannelSetupEntry(t, plan, channel)
		rendered := strings.Join(append(append([]string{}, entry.RequiredFields...), entry.PlannedWrites...), "\n")
		for _, want := range wants {
			if !strings.Contains(rendered, want) {
				t.Fatalf("%s setup guidance missing %q:\n%s", channel, want, rendered)
			}
		}
	}
}

func TestProfileChannelSetupPlanWhatsAppUsesReadinessAndRedactsScopes(t *testing.T) {
	cfg := config.Config{
		Profiles: map[string]config.ProfileCfg{
			"main": {
				Enabled: true,
				Channels: map[string]config.ProfileChannelCfg{
					"whatsapp": {
						Enabled:      true,
						Credential:   "main-whatsapp",
						AllowedChats: []string{"12025550123@s.whatsapp.net"},
						AllowedUsers: []string{"6586915095"},
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

	plan := BuildChannelSetupPlan(cfg)
	whatsapp := findChannelSetupEntry(t, plan, "whatsapp")
	if whatsapp.Status != ChannelSetupStatusConfigured {
		t.Fatalf("whatsapp status = %q, want configured for ready profile bindings: %+v", whatsapp.Status, whatsapp)
	}
	rendered := strings.Join(append(append([]string{}, whatsapp.CurrentValues...), whatsapp.Warnings...), "\n")
	for _, want := range []string{
		"profiles.main.channels.whatsapp.credential=main-whatsapp",
		"profiles.main.channels.whatsapp.allowed_chats=1",
		"profiles.main.channels.whatsapp.allowed_users=1",
		"profiles.sales.channels.whatsapp.credential=sales-whatsapp",
		"profiles.sales.channels.whatsapp.allowed_chats=1",
		"profiles.sales.channels.whatsapp.allowed_users=1",
		"credentials.main-whatsapp.secret_ref=[REDACTED:env]",
		"credentials.sales-whatsapp.secret_ref=[REDACTED:env]",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("whatsapp setup values missing %q:\n%s", want, rendered)
		}
	}
	for _, leaked := range []string{
		"12025550123", "12025550999", "6586915095", "7770001111",
		"GORMES_MAIN_WHATSAPP_TOKEN", "GORMES_SALES_WHATSAPP_TOKEN",
	} {
		if strings.Contains(rendered, leaked) {
			t.Fatalf("whatsapp setup plan leaked sensitive value %q:\n%s", leaked, rendered)
		}
	}
}

func TestProfileChannelSetupPlanWhatsAppReportsDuplicateTokenHashConflict(t *testing.T) {
	const rawSharedToken = "123456:shared-whatsapp-token-that-must-not-leak"
	sharedHash := TokenCredentialHash(rawSharedToken)
	cfg := config.Config{
		Profiles: map[string]config.ProfileCfg{
			"main": {
				Enabled: true,
				Channels: map[string]config.ProfileChannelCfg{
					"whatsapp": {
						Enabled:      true,
						Credential:   "main-whatsapp",
						AllowedChats: []string{"12025550123@s.whatsapp.net"},
						AllowedUsers: []string{"6586915095"},
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
			"main-whatsapp":  channelCredential("whatsapp", "main", "GORMES_MAIN_WHATSAPP_TOKEN"),
			"sales-whatsapp": channelCredential("whatsapp", "sales", "GORMES_SALES_WHATSAPP_TOKEN"),
		},
	}

	plan := BuildChannelSetupPlanWithOptions(cfg, ChannelSetupPlanOptions{
		CredentialHashes: map[string]string{
			"main-whatsapp":  sharedHash,
			"sales-whatsapp": sharedHash,
		},
	})
	whatsapp := findChannelSetupEntry(t, plan, "whatsapp")
	if whatsapp.Status != ChannelSetupStatusPartial {
		t.Fatalf("whatsapp status = %q, want partial when two profiles reuse one WhatsApp token hash: %+v", whatsapp.Status, whatsapp)
	}
	rendered := strings.Join(append(append([]string{}, whatsapp.CurrentValues...), whatsapp.Warnings...), "\n")
	for _, want := range []string{
		"profiles.main.channels.whatsapp: channel_token_hash_conflict (credential_hash)",
		"profiles.sales.channels.whatsapp: channel_token_hash_conflict (credential_hash)",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("whatsapp setup duplicate-token guidance missing %q:\n%s", want, rendered)
		}
	}
	for _, forbidden := range []string{rawSharedToken, sharedHash, "12025550123", "12025550999", "6586915095", "7770001111", "GORMES_MAIN_WHATSAPP_TOKEN", "GORMES_SALES_WHATSAPP_TOKEN"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("whatsapp setup duplicate-token guidance leaked sensitive value %q:\n%s", forbidden, rendered)
		}
	}
}

func TestProfileChannelSetupPlanWhatsAppNormalizesProfileChannelKey(t *testing.T) {
	cfg := config.Config{
		Profiles: map[string]config.ProfileCfg{
			"main": {
				Enabled: true,
				Channels: map[string]config.ProfileChannelCfg{
					"WhatsApp": {
						Enabled:      true,
						Credential:   "main-whatsapp",
						AllowedChats: []string{"12025550123@s.whatsapp.net"},
						AllowedUsers: []string{"6586915095"},
					},
				},
			},
		},
		Credentials: map[string]config.CredentialCfg{
			"main-whatsapp": {
				Kind:         "channel",
				Channel:      "WhatsApp",
				OwnerProfile: "main",
				SecretRef: &config.SecretRef{
					Source: config.SecretRefSourceEnv,
					ID:     "GORMES_MAIN_WHATSAPP_TOKEN",
				},
			},
		},
	}

	plan := BuildChannelSetupPlan(cfg)
	whatsapp := findChannelSetupEntry(t, plan, "whatsapp")
	if whatsapp.Status != ChannelSetupStatusConfigured {
		t.Fatalf("whatsapp status = %q, want configured for mixed-case profile channel key: %+v", whatsapp.Status, whatsapp)
	}
	rendered := strings.Join(append(append([]string{}, whatsapp.CurrentValues...), whatsapp.Warnings...), "\n")
	for _, want := range []string{
		"profiles.main.channels.whatsapp.credential=main-whatsapp",
		"profiles.main.channels.whatsapp.allowed_chats=1",
		"profiles.main.channels.whatsapp.allowed_users=1",
		"credentials.main-whatsapp.secret_ref=[REDACTED:env]",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("whatsapp setup values missing %q:\n%s", want, rendered)
		}
	}
	for _, forbidden := range []string{"channel_credential_missing", "GORMES_MAIN_WHATSAPP_TOKEN", "12025550123", "6586915095"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("whatsapp setup values contain forbidden %q:\n%s", forbidden, rendered)
		}
	}
}

func TestProfileChannelSetupPlanWhatsAppReportsPairedLoginStatus(t *testing.T) {
	cfg := config.Config{
		Profiles: map[string]config.ProfileCfg{
			"main": {
				Enabled: true,
				Channels: map[string]config.ProfileChannelCfg{
					"whatsapp": {
						Enabled:      true,
						Credential:   "main-whatsapp",
						AllowedChats: []string{"12025550123@s.whatsapp.net"},
						AllowedUsers: []string{"6586915095"},
					},
				},
			},
		},
		Credentials: map[string]config.CredentialCfg{
			"main-whatsapp": channelCredential("whatsapp", "main", "GORMES_MAIN_WHATSAPP_TOKEN"),
		},
	}

	plan := BuildChannelSetupPlanWithOptions(cfg, ChannelSetupPlanOptions{
		Pairing: PairingStatus{
			Platforms: []PairingPlatformStatus{{
				Platform:      "whatsapp",
				State:         PairingPlatformStatePaired,
				PendingCount:  2,
				ApprovedCount: 1,
			}},
		},
	})
	whatsapp := findChannelSetupEntry(t, plan, "whatsapp")
	if whatsapp.Status != ChannelSetupStatusPaired {
		t.Fatalf("whatsapp status = %q, want paired after configured profile binding and paired login state: %+v", whatsapp.Status, whatsapp)
	}
	if whatsapp.NextCommand != "gormes gateway" {
		t.Fatalf("whatsapp next command = %q, want gateway start after paired login", whatsapp.NextCommand)
	}
	rendered := strings.Join(append(append([]string{}, whatsapp.CurrentValues...), whatsapp.Warnings...), "\n")
	for _, want := range []string{
		"whatsapp.pairing=paired",
		"whatsapp.pairing_approved_users=1",
		"whatsapp.pairing_pending_codes=2",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("whatsapp setup values missing %q:\n%s", want, rendered)
		}
	}
	for _, forbidden := range []string{"12025550123", "6586915095", "GORMES_MAIN_WHATSAPP_TOKEN"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("whatsapp setup login values leaked sensitive value %q:\n%s", forbidden, rendered)
		}
	}
}

func TestProfileChannelSetupPlanWhatsAppReportsUnpairedLoginStatus(t *testing.T) {
	cfg := config.Config{
		Profiles: map[string]config.ProfileCfg{
			"main": {
				Enabled: true,
				Channels: map[string]config.ProfileChannelCfg{
					"whatsapp": {
						Enabled:      true,
						Credential:   "main-whatsapp",
						AllowedChats: []string{"12025550123@s.whatsapp.net"},
						AllowedUsers: []string{"6586915095"},
					},
				},
			},
		},
		Credentials: map[string]config.CredentialCfg{
			"main-whatsapp": channelCredential("whatsapp", "main", "GORMES_MAIN_WHATSAPP_TOKEN"),
		},
	}

	plan := BuildChannelSetupPlanWithOptions(cfg, ChannelSetupPlanOptions{
		Pairing: PairingStatus{
			Platforms: []PairingPlatformStatus{{
				Platform:      "whatsapp",
				State:         PairingPlatformStateUnpaired,
				PendingCount:  1,
				ApprovedCount: 0,
			}},
		},
	})
	whatsapp := findChannelSetupEntry(t, plan, "whatsapp")
	if whatsapp.Status != ChannelSetupStatusPartial {
		t.Fatalf("whatsapp status = %q, want partial until WhatsApp pairing is complete: %+v", whatsapp.Status, whatsapp)
	}
	if whatsapp.NextCommand != "gormes whatsapp" {
		t.Fatalf("whatsapp next command = %q, want live pairing wizard", whatsapp.NextCommand)
	}
	rendered := strings.Join(append(append([]string{}, whatsapp.CurrentValues...), whatsapp.Warnings...), "\n")
	for _, want := range []string{
		"whatsapp.pairing=unpaired",
		"whatsapp.pairing_approved_users=0",
		"whatsapp.pairing_pending_codes=1",
		"WhatsApp pairing is not complete",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("whatsapp setup values missing %q:\n%s", want, rendered)
		}
	}
	for _, forbidden := range []string{"12025550123", "6586915095", "GORMES_MAIN_WHATSAPP_TOKEN"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("whatsapp setup login values leaked sensitive value %q:\n%s", forbidden, rendered)
		}
	}
}

func TestProfileChannelSetupPlanWhatsAppReportsPairingDegradedStatus(t *testing.T) {
	cfg := config.Config{
		Profiles: map[string]config.ProfileCfg{
			"main": {
				Enabled: true,
				Channels: map[string]config.ProfileChannelCfg{
					"whatsapp": {
						Enabled:      true,
						Credential:   "main-whatsapp",
						AllowedChats: []string{"12025550123@s.whatsapp.net"},
						AllowedUsers: []string{"6586915095"},
					},
				},
			},
		},
		Credentials: map[string]config.CredentialCfg{
			"main-whatsapp": channelCredential("whatsapp", "main", "GORMES_MAIN_WHATSAPP_TOKEN"),
		},
	}

	plan := BuildChannelSetupPlanWithOptions(cfg, ChannelSetupPlanOptions{
		Pairing: PairingStatus{
			Degraded: []PairingDegradedEvidence{{
				Platform: "whatsapp",
				Reason:   PairingDegradedPermissionDenied,
				Path:     "/home/xel/.gormes/pairing.json",
				UserID:   "6586915095",
				Code:     "PAIR1234",
				Message:  "read /home/xel/.gormes/pairing.json: permission denied",
			}},
		},
	})
	whatsapp := findChannelSetupEntry(t, plan, "whatsapp")
	if whatsapp.Status != ChannelSetupStatusPartial {
		t.Fatalf("whatsapp status = %q, want partial when pairing readout is degraded: %+v", whatsapp.Status, whatsapp)
	}
	if whatsapp.NextCommand != "gormes whatsapp" {
		t.Fatalf("whatsapp next command = %q, want live pairing wizard for degraded pairing readout", whatsapp.NextCommand)
	}
	rendered := strings.Join(append(append(append([]string{}, whatsapp.CurrentValues...), whatsapp.Warnings...), whatsapp.PlannedWrites...), "\n")
	for _, want := range []string{
		"whatsapp.pairing_degraded=permission_denied",
		"WhatsApp pairing status is degraded: permission_denied",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("whatsapp setup degraded guidance missing %q:\n%s", want, rendered)
		}
	}
	for _, forbidden := range []string{"/home/xel/.gormes/pairing.json", "PAIR1234", "6586915095", "GORMES_MAIN_WHATSAPP_TOKEN"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("whatsapp setup degraded guidance leaked sensitive value %q:\n%s", forbidden, rendered)
		}
	}
}

func TestProfileChannelSetupPlanWhatsAppSeparatesGroupAndDirectAllowedChats(t *testing.T) {
	cfg := config.Config{
		Profiles: map[string]config.ProfileCfg{
			"main": {
				Enabled: true,
				Channels: map[string]config.ProfileChannelCfg{
					"whatsapp": {
						Enabled:      true,
						Credential:   "main-whatsapp",
						AllowedChats: []string{"12025550123@s.whatsapp.net", "12025550999-123@g.us"},
						AllowedUsers: []string{"6586915095"},
					},
				},
			},
		},
		Credentials: map[string]config.CredentialCfg{
			"main-whatsapp": channelCredential("whatsapp", "main", "GORMES_MAIN_WHATSAPP_TOKEN"),
		},
	}

	plan := BuildChannelSetupPlan(cfg)
	whatsapp := findChannelSetupEntry(t, plan, "whatsapp")
	rendered := strings.Join(whatsapp.CurrentValues, "\n")
	for _, want := range []string{
		"profiles.main.channels.whatsapp.allowed_chats=2",
		"profiles.main.channels.whatsapp.allowed_direct_chats=1",
		"profiles.main.channels.whatsapp.allowed_group_chats=1",
		"profiles.main.channels.whatsapp.allowed_users=1",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("whatsapp setup values missing %q:\n%s", want, rendered)
		}
	}
	for _, leaked := range []string{"12025550123", "12025550999", "6586915095"} {
		if strings.Contains(rendered, leaked) {
			t.Fatalf("whatsapp setup plan leaked sensitive value %q:\n%s", leaked, rendered)
		}
	}
}

func TestProfileChannelSetupPlanWhatsAppCanonicalizesAllowedUserJIDForms(t *testing.T) {
	cfg := config.Config{
		Profiles: map[string]config.ProfileCfg{
			"main": {
				Enabled: true,
				Channels: map[string]config.ProfileChannelCfg{
					"whatsapp": {
						Enabled:      true,
						Credential:   "main-whatsapp",
						AllowedChats: []string{"12025550123@s.whatsapp.net"},
						AllowedUsers: []string{" +6586915095 ", "6586915095@s.whatsapp.net", "6586915095:47@s.whatsapp.net"},
					},
				},
			},
		},
		Credentials: map[string]config.CredentialCfg{
			"main-whatsapp": channelCredential("whatsapp", "main", "GORMES_MAIN_WHATSAPP_TOKEN"),
		},
	}

	plan := BuildChannelSetupPlan(cfg)
	whatsapp := findChannelSetupEntry(t, plan, "whatsapp")
	if whatsapp.Status != ChannelSetupStatusConfigured {
		t.Fatalf("whatsapp status = %q, want configured after canonicalizing WhatsApp user ids: %+v", whatsapp.Status, whatsapp)
	}
	rendered := strings.Join(append(append([]string{}, whatsapp.CurrentValues...), whatsapp.Warnings...), "\n")
	if !strings.Contains(rendered, "profiles.main.channels.whatsapp.allowed_users=1") {
		t.Fatalf("whatsapp setup allowed user count missing canonicalized value:\n%s", rendered)
	}
	for _, forbidden := range []string{"+6586915095", "6586915095@s.whatsapp.net", "6586915095:47@s.whatsapp.net", "GORMES_MAIN_WHATSAPP_TOKEN"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("whatsapp setup allowed-user guidance leaked sensitive value %q:\n%s", forbidden, rendered)
		}
	}
}

func TestProfileChannelSetupPlanWhatsAppRequiresAccessPolicy(t *testing.T) {
	cfg := config.Config{
		Profiles: map[string]config.ProfileCfg{
			"main": {
				Enabled: true,
				Channels: map[string]config.ProfileChannelCfg{
					"whatsapp": {
						Enabled:    true,
						Credential: "main-whatsapp",
					},
				},
			},
		},
		Credentials: map[string]config.CredentialCfg{
			"main-whatsapp": channelCredential("whatsapp", "main", "GORMES_MAIN_WHATSAPP_TOKEN"),
		},
	}

	plan := BuildChannelSetupPlan(cfg)
	whatsapp := findChannelSetupEntry(t, plan, "whatsapp")
	if whatsapp.Status != ChannelSetupStatusPartial {
		t.Fatalf("whatsapp status = %q, want partial when credential lacks chat/user access policy: %+v", whatsapp.Status, whatsapp)
	}
	rendered := strings.Join(append(append([]string{}, whatsapp.Warnings...), whatsapp.PlannedWrites...), "\n")
	for _, want := range []string{
		"profiles.main.channels.whatsapp: channel_access_policy_missing (allowed_chats)",
		"profiles.main.channels.whatsapp.allowed_chats or allowed_users -> config.toml",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("whatsapp setup guidance missing %q:\n%s", want, rendered)
		}
	}
	if strings.Contains(rendered, "GORMES_MAIN_WHATSAPP_TOKEN") {
		t.Fatalf("whatsapp setup guidance leaked secret ref id:\n%s", rendered)
	}
}

func TestChannelSetupPlanListsMessagingPlatforms(t *testing.T) {
	plan := BuildChannelSetupPlan(config.Config{})
	for _, want := range []string{"telegram", "discord", "slack", "whatsapp", "navivox"} {
		entry := findChannelSetupEntry(t, plan, want)
		if entry.Status != ChannelSetupStatusUnconfigured {
			t.Fatalf("%s status = %q, want unconfigured in empty config", want, entry.Status)
		}
		if len(entry.RequiredFields) == 0 {
			t.Fatalf("%s RequiredFields empty, want setup guidance", want)
		}
	}
}

func findChannelSetupEntry(t *testing.T, plan ChannelSetupPlan, id string) ChannelSetupEntry {
	t.Helper()
	for _, entry := range plan.Channels {
		if entry.ID == id {
			return entry
		}
	}
	t.Fatalf("missing channel setup entry %q in %+v", id, plan.Channels)
	return ChannelSetupEntry{}
}
